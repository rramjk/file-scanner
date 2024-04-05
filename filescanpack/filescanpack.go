package filescanpack

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type Stringer interface {
	String() string
}

// FileInfo - объект файловой системы
type FileInfo struct {
	// название элемента
	Name string
	// размера элемента в байтах
	Size int64
	// папка или файл (true or false)
	IsDir bool
}

func (f FileInfo) String() string {
	if f.IsDir {
		return fmt.Sprintf("Папка | %s | %s", f.Name, FormatSize(f.Size))
	} else {
		return fmt.Sprintf("Файл | %s | %s", f.Name, FormatSize(f.Size))
	}
}

// getDirectoryContents - собирает все вложенные элементы по пути указанному в параметр
func GetDirectoryContents(dirPath string) ([]FileInfo, error) {
	var fileList []FileInfo

	// Читаем содержимое директории
	files, err := os.ReadDir(dirPath)
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
		go func(file os.DirEntry) {
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
func calculateSize(path string, inf os.DirEntry) (int64, error) {
	info, err := inf.Info()
	if err != nil {
		return 0, err
	}

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

// добавить перевод файлов в кб мб гб
// formatSize - конвертирует размер из в байт в более понятную систему счисления
func FormatSize(size int64) string {
	n := float64(size)
	i := 0

	for n >= 1024 {
		i++
		n /= 1024
	}
	switch {
	case i == 1:
		return fmt.Sprintf("%.2f KB", n)
	case i == 2:
		return fmt.Sprintf("%.2f MB", n)
	case i == 3:
		return fmt.Sprintf("%.2f GB", n)
	default:
		return fmt.Sprintf("%d bytes", size)
	}
}
