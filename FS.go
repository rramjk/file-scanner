package main

import (
	"errors"
	"flag"
	"fmt"
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

	// путь для предполагаемогой дирректории
	var directorySource string
	var sortBy string
	// получение параметров из командной строки
	err := parseParam(&directorySource, &sortBy)

	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
	// вывести отображение файлов
	err = printFiles(directorySource, sortBy)
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	fmt.Printf("Время работы программы: %v\n", time.Now().Sub(startTime))
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
func printFiles(directorySource string, sortBy string) error {
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
	for _, file := range files {
		fmt.Printf("%s | %s | %s\n", file.Type, file.Name, mustFormatSize(file.Size))
	}
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
