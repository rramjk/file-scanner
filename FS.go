package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type FileInfo struct {
	Name  string
	Size  int64
	IsDir bool
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

// filesHandler - обработчик для GET-запросов
func filesHandler(w http.ResponseWriter, r *http.Request) {
	// Проверяем, что запрос является GET-запросом
	if r.Method != http.MethodGet {
		http.Error(w, "Метод запрещен.", http.StatusMethodNotAllowed)
		return
	}

	// Получаем параметры запроса
	rootParam := r.URL.Query().Get("root")
	sortParam := r.URL.Query().Get("sort")

	if rootParam == "" {
		fmt.Fprint(w, "Введите параметры root and sort! ?root=&sort=(ASC default)")
		return
	}
	if sortParam == rootParam || sortParam == "" {
		sortParam = "ASC"
	}
	err := convertAndSendFilesIntoRootToServer(&w, rootParam, sortParam)
	if err != nil {
		fmt.Println(err)
		return
	}

}

/*
convertAndSendFilesIntoRootToServer - данный метод получает на вход путь к директории и параметр сортировки
после чего получает все элементы в папке, сортирует их и отправляет их на сервер в формате JSON
*/
func convertAndSendFilesIntoRootToServer(w *http.ResponseWriter, dirSource string, sort string) error {
	// путь для предполагаемогой дирректории
	directorySource := dirSource
	sortBy := sort

	// вывести отображение файлов
	fileList, err := getDirectoryContents(directorySource)
	if err != nil {
		return err
	}
	mustSortDirectoryContents(fileList, sortBy)
	err = sendJsonViewOnServer(w, fileList)
	if err != nil {
		return err
	}
	return nil
}

// getDirectoryContents - собирает все вложенные элементы по пути указанному в параметр
func getDirectoryContents(dirPath string) ([]FileInfo, error) {
	var fileList []FileInfo

	// Читаем содержимое директории
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	// Проходим по файлам и папкам верхнего уровня
	for _, file := range files {
		fileSize, err := calculateSize(filepath.Join(dirPath, file.Name()), file)
		if err != nil {
			return nil, err
		}
		fileList = append(fileList, FileInfo{
			Name:  file.Name(),
			Size:  fileSize,
			IsDir: file.IsDir(),
		})
	}

	return fileList, nil
}

// calculateSize - подсчитывает размер файла или вложенных в папку элементов
func calculateSize(path string, info os.FileInfo) (int64, error) {
	if !info.IsDir() {
		return info.Size(), nil
	}

	var size int64
	// Используем filepath.Walk для вычисления размера папки
	_ = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		size += info.Size()
		return nil
	})

	return size, nil
}

// mustSortDirectoryContents - метод сортирует полученные элементы согласно параметру сортировки
func mustSortDirectoryContents(fileList []FileInfo, sortBy string) {
	sort.Slice(fileList, func(i, j int) bool {
		if sortBy == "ASC" {
			return fileList[i].Size < fileList[j].Size
		} else {
			return fileList[i].Size > fileList[j].Size
		}
	})
}

// formatSize - конвертирует размер из в байт в более понятную систему счисления
func formatSize(size int64) string {
	const (
		KB = 1 << 10
		MB = 1 << 20
		GB = 1 << 30
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d bytes", size)
	}
}

// sendJsonViewOnServer - метод выводит данные для проверки и отправляет их на сервер в формате JSON
func sendJsonViewOnServer(w *http.ResponseWriter, fileList []FileInfo) error {
	// var wg sync.WaitGroup
	// var mu sync.Mutex

	for _, fileInfo := range fileList {
		itemType := "файл"
		if fileInfo.IsDir {
			itemType = "папка"
		}
		fmt.Printf("%s | %s | %s\n", itemType, fileInfo.Name, formatSize(fileInfo.Size))
	}
	err := sendJson(w, &fileList)
	if err != nil {
		return err
	}
	return nil
}

// sendJson - отправляет данные на сервер в формате JSON
func sendJson(w *http.ResponseWriter, files *[]FileInfo) error {
	jsonData, err := json.Marshal(files)
	if err != nil {
		return err
	}
	(*w).Header().Set("Content-Type", "application/json")
	(*w).Write(jsonData)

	return nil
}
