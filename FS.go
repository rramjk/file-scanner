package main

import (
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type FileInfo struct {
	Type string
	Name string
	Size int64
	Path string
}

var noSuchDirectoryError = errors.New("Директории не существует")

func main() {
	// получаем время начала программы
	startTime := time.Now()

	// путь для предполагаемого файла и дирректории
	var directorySource string
	var sortBy string

	err := parseParam(&directorySource, &sortBy)
	//
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
	printFiles(directorySource, sortBy)

	fmt.Printf("Время работы программы: %v\n", time.Now().Sub(startTime))
}

// parseParam - получение параметров с вызова программы
func parseParam(directorySource *string, sortBy *string) error {
	flag.StringVar(directorySource, "root", "null", "source of directory")
	flag.StringVar(sortBy, "sort", "ASC", "param from sort")
	flag.Parse()

	// проверка на корректность полученных параметров
	err := DirectorySrcAndSortedParamIsCorrect(*directorySource, *sortBy)
	if err != nil {
		return err
	}
	return nil
}

// srcAndDstIsCorrect - проверка источника и необходимой дирректории на корректность (true - src: удачный путь к файлу dst: корректная папка)
func DirectorySrcAndSortedParamIsCorrect(directorySource string, sortBy string) error {
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

func addInnerEntityFromDirectory(dir string, files *[]FileInfo, wg *sync.WaitGroup, mu *sync.Mutex) {
	defer wg.Done()

	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			fmt.Println(err)
			continue
		}

		if entry.IsDir() {
			wg.Add(1)
			go getDirSize(filepath.Join(dir, entry.Name()), files, wg, mu)
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
}

func getSumBytesfromChannel(ch *chan int64) int64 {
	var sum int64 = 0
	for i := 0; i < len(*ch); i++ {
		sum += <-*ch
	}
	return sum
}
func getDirSize(dir string, files *[]FileInfo, wg *sync.WaitGroup, mu *sync.Mutex) {
	defer wg.Done()

	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Println(err)
		return
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
			go getDirSize(filepath.Join(dir, entry.Name()), files, wg, mu)
		} else {
			size += info.Size()
		}
	}

	mu.Lock()
	*files = append(*files, FileInfo{
		Type: "папка",
		Name: filepath.Base(dir),
		Size: size,
		Path: dir,
	})
	mu.Unlock()
}

func sortFiles(files []FileInfo, sortBy string) {
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
		fmt.Println("Invalid sortBy value")
	}
}

func printFiles(directorySource string, sortBy string) {
	var files []FileInfo
	var wg sync.WaitGroup
	var mu sync.Mutex

	wg.Add(1)
	go addInnerEntityFromDirectory(directorySource, &files, &wg, &mu)

	wg.Wait()
	for _, file := range files {
		fmt.Printf("%s | %s | %s\n", file.Type, file.Name, formatSize(file.Size))
	}
	fmt.Println("--------SORTED--------")
	sortFiles(files, sortBy)
	for _, file := range files {
		fmt.Printf("%s | %s | %s\n", file.Type, file.Name, formatSize(file.Size))
	}
}
func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/math.Pow(float64(unit), float64(exp)), " KMGTPE"[exp])
}
