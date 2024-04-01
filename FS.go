package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// единичный элемент, либо папка либо файл
type FileInfo struct {
	// тип файл или папка
	Type string
	// название элемента
	Name string
	// размер элемента
	Size int64
	// путь к элементу
	Path string
}

var noSuchDirectoryError = errors.New("Директории не существует")

func main() {
	// получаем время начала программы
	startTime := time.Now()
	// Устанавливаем обработчик для корневого пути
	http.HandleFunc("/files", filesHandler)

	// Запускаем HTTP-сервер на порту 8080
	fmt.Println("Запуск сервера на http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Ошибка при запуске сервера: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Время работы программы: %v\n", time.Now().Sub(startTime))
}

// Обработчик для GET-запросов
func filesHandler(w http.ResponseWriter, r *http.Request) {
	// Проверяем, что запрос является GET-запросом
	if r.Method != http.MethodGet {
		http.Error(w, "Метод запрещен.", http.StatusMethodNotAllowed)
		return
	}

	// Получаем параметры запроса
	rootParam := r.URL.Query().Get("root")
	sortParam := r.URL.Query().Get("sort")
	fmt.Print(fmt.Sprintf("%s %s", rootParam, sortParam))
	if rootParam == "" {
		fmt.Fprint(w, "Введите параметры root and sort! ?root=&sort=(ASC default)")
	} else {
		if sortParam == rootParam || sortParam == "" {
			sortParam = "ASC"
		}
		sortParam = "ASC"
		fmt.Println(sortParam)
		err := showFile(&w, rootParam, sortParam)

		if err != nil {
			fmt.Println(err)
		}
	}
}

func showFile(w *http.ResponseWriter, dirSource string, sort string) error {
	// путь для предполагаемогой дирректории
	directorySource := dirSource
	sortBy := sort
	fmt.Print(fmt.Sprintf("%s %s", directorySource, sortBy))
	// вывести отображение файлов
	err := printFiles(w, directorySource, sortBy)
	if err != nil {
		return err
	}
	return nil
}

// parseParam - получение параметров с вызова программы
func parseParam(directorySource *string, sortBy *string) error {
	flag.StringVar(directorySource, "root", "null", "source of directory")
	flag.StringVar(sortBy, "sort", "ASC", "param from sort")
	flag.Parse()

	// проверка на корректность полученных параметров
	err := directorySrcAndSortedParamIsCorrect(*directorySource, *sortBy)
	if err != nil {
		return err
	}
	return nil
}

// directorySrcAndSortedParamIsCorrect - проверка необходимой дирректории на корректность
// (true - root: удачный путь к файлу sort: корректная папка)
func directorySrcAndSortedParamIsCorrect(directorySource string, sortBy string) error {
	if directorySource == "null" || (sortBy != "ASC" && sortBy != "DESC") {
		return errors.New("Источник или параметр сортировки указаны не верно\n./FS --root='path to directory' --sort='param of sort (default ASC)'")
	}
	// тестовое получение папки если оно удачно значит создавать новую папку не стоит
	drtInfo, drtErr := os.Stat(directorySource)
	if (directorySource[:1] == "/" || directorySource[:2] == "./") && drtErr != nil {
		return noSuchDirectoryError
	}
	if drtErr != nil {
		if os.IsNotExist(drtErr) {
			return errors.New("Директория не существует")
		} else {
			return errors.New("Ошибка получения информации о директории")
		}
	}
	if !(drtInfo.IsDir()) {
		return errors.New("Параметр --root должен содержать путь к папке")
	}
	return nil
}

// addInnerEntityFromDirectory - метод заполняет срез объектов элементами дирректории(файлами папками)
func addInnerEntityFromDirectory(dir string, files *[]FileInfo, wg *sync.WaitGroup, mu *sync.Mutex) error {
	defer wg.Done()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			fmt.Println(err)
			continue
		}
		level := 1
		if entry.IsDir() {
			wg.Add(1)
			go addFolderWithFullInnerElementSize(filepath.Join(dir, entry.Name()), files, wg, mu, level)
		} else {
			mu.Lock()
			*files = append(*files, FileInfo{
				Type: "файл",
				Name: entry.Name(),
				Size: info.Size(),
				Path: filepath.Join(dir, entry.Name()),
			})
			mu.Unlock()
		}
	}
	return nil
}

// addFolderWithFullInnerElementSize - данный метод является инструкцией для потока.
// Метод проходится по полученной дирректории, если это первый уровень вложенности, то сразу добавляет дирректорию в срез, иначе он лишь
// суммирует к размеру дирректории размер файлов вложенных в нее
func addFolderWithFullInnerElementSize(dir string, files *[]FileInfo, wg *sync.WaitGroup, mu *sync.Mutex, level int) error {
	defer wg.Done()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	var size int64 = 0
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			fmt.Println(err)
			continue
		}

		if entry.IsDir() {
			wg.Add(1)
			go addFolderWithFullInnerElementSize(filepath.Join(dir, entry.Name()), files, wg, mu, level+1)
		} else {
			size += info.Size()
		}
	}
	if level == 1 {
		mu.Lock()
		*files = append(*files, FileInfo{
			Type: "папка",
			Name: filepath.Base(dir),
			Size: size,
			Path: dir,
		})
		mu.Unlock()
	} else {
		mu.Lock()
		(*files)[len((*files))-1].Size += size
		mu.Unlock()
	}
	return nil
}

// sortFiles - метод который по принципу описания анонимной функции в sort.Slice описывается поведения для меньшего элемента
func sortFiles(files []FileInfo, sortBy string) error {
	switch sortBy {
	case "ASC":
		sort.Slice(files, func(i, j int) bool {
			return files[i].Size < files[j].Size
		})
	case "DESC":
		sort.Slice(files, func(i, j int) bool {
			return files[i].Size > files[j].Size
		})
	default:
		return errors.New("Invalid sortBy value")
	}
	return nil
}

// printFiles - метод вывода по шаблону заполненного массива, прошедшего сортировку
func printFiles(w *http.ResponseWriter, directorySource string, sortBy string) error {
	var files []FileInfo
	var wg sync.WaitGroup
	var mu sync.Mutex

	wg.Add(1)
	go addInnerEntityFromDirectory(directorySource, &files, &wg, &mu)
	wg.Wait()
	err := sortFiles(files, sortBy)
	if err != nil {
		return err
	}

	sendJson(w, &files)
	return nil

}

// formatSize - метод, который переводит байты в понятные единицы измерения гб, мб, кб
// метод определяет сколько раз кол-во байт можно поделить на 1024 тем самым определяет ед. измерения
func mustFormatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	n, exp := float64(size), 0
	for n > 1024 {
		n /= 1024
		exp++
	}
	return fmt.Sprintf("%.1f %cB", n, " KMGTPE"[exp])
}

func sendJson(w *http.ResponseWriter, files *[]FileInfo) error {
	jsonData, err := json.Marshal(files)
	if err != nil {
		return err
	}
	(*w).Header().Set("Content-Type", "application/json")
	(*w).Write(jsonData)

	return nil
}
