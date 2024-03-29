package main

import (
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
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

func walkDir(dir string, files *[]FileInfo) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, entry := range entries {
		_, err := entry.Info()
		if err != nil {
			fmt.Println(err)
			continue
		}

		if entry.IsDir() {
			filesSize := getDirSize(filepath.Join(dir, entry.Name()))
			*files = append(*files, FileInfo{
				Type: "папка",
				Name: entry.Name(),
				Size: filesSize,
				Path: filepath.Join(dir, entry.Name()),
			})
		} else {
			filesSize, err := os.Stat(filepath.Join(dir, entry.Name()))
			if err != nil {
				fmt.Println(err)
			}
			*files = append(*files, FileInfo{
				Type: "файл",
				Name: entry.Name(),
				Size: filesSize.Size(),
				Path: filepath.Join(dir, entry.Name()),
			})
		}
	}
}

func getDirSize(dir string) int64 {
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Println(err)
		return 0
	}

	var size int64 = 0
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			fmt.Println(err)
			continue
		}

		if entry.IsDir() {
			size += getDirSize(filepath.Join(dir, entry.Name()))
		} else {
			size += info.Size()
		}
	}
	return size
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
	files := []FileInfo{}
	walkDir(directorySource, &files)
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
