package filescanpack

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type FileInfo struct {
	Name  string
	Size  int64
	IsDir bool
}

// getDirectoryContents - собирает все вложенные элементы по пути указанному в параметр
func GetDirectoryContents(dirPath string) ([]FileInfo, error) {
	var fileList []FileInfo

	// Читаем содержимое директории
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	// Создаем WaitGroup для ожидания завершения всех горутин
	var wg sync.WaitGroup

	// Создаем мьютекс для безопасного доступа к fileList
	var mu sync.Mutex

	// Проходим по файлам и папкам верхнего уровня
	for _, file := range files {
		wg.Add(1)
		go func(file os.FileInfo) {
			defer wg.Done()
			fileSize, err := calculateSize(filepath.Join(dirPath, file.Name()), file)
			if err != nil {
				fmt.Printf("Ошибка при вычислении размера: %v\n", err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			fileList = append(fileList, FileInfo{
				Name:  file.Name(),
				Size:  fileSize,
				IsDir: file.IsDir(),
			})
		}(file)
	}

	// Ждем, пока все горутины завершат свою работу
	wg.Wait()

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
func MustSortDirectoryContents(fileList []FileInfo, sortBy string) {
	sort.Slice(fileList, func(i, j int) bool {
		if sortBy == "ASC" {
			return fileList[i].Size < fileList[j].Size
		} else {
			return fileList[i].Size > fileList[j].Size
		}
	})
}

// formatSize - конвертирует размер из в байт в более понятную систему счисления
func FormatSize(size int64) string {
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
