package main

import (
	"context"
	"encoding/json"
	"errors"
	"filescanpack/filescanpack"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	// получаем время начала программы
	startTime := time.Now()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()
	mux.HandleFunc("/files", filesHandler)
	mux.Handle("/", http.FileServer(http.Dir("./ui")))
	mux.Handle("/ui/", http.StripPrefix("/ui/", http.FileServer(http.Dir("./ui"))))

	port, err := getPort()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Запускаем HTTP-сервер на порту 8080
	fmt.Println("Запуск сервера на http://localhost:" + port)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%v", port),
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Ошибка при заупуске сервера: %s\n", err)
			stop()
		}

	}()
	<-ctx.Done()

	fmt.Println("\nЗавершаем работу сервера...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("Ошибка при завершени работы сервера: %s\n", err)
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
		rootParam = "/home"
	}
	if sortParam == rootParam || sortParam == "" {
		sortParam = "ASC"
	}
	fileList, err := getSortedFilesIntoRoot(rootParam, sortParam)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = sendJsonViewOnServer(&w, fileList)
	if err != nil {
		fmt.Println(err)
		return
	}

}

/*
convertAndSendFilesIntoRootToServer - данный метод получает на вход путь к директории и параметр сортировки
после чего получает все элементы в папке, сортирует их и отправляет их на сервер в формате JSON
*/
func getSortedFilesIntoRoot(dirSource string, sort string) ([]filescanpack.FileInfo, error) {
	// путь для предполагаемогой дирректории
	directorySource := dirSource
	sortBy := sort
	// вывести отображение файлов
	fileList, err := filescanpack.GetDirectoryContents(directorySource)

	if err != nil {
		return nil, err
	}
	filescanpack.MustSortDirectoryContents(fileList, sortBy)

	return fileList, nil
}

// sendJsonViewOnServer - метод выводит данные для проверки и отправляет их на сервер в формате JSON
func sendJsonViewOnServer(w *http.ResponseWriter, fileList []filescanpack.FileInfo) error {
	for _, fileInfo := range fileList {
		fmt.Println(fileInfo.String())
	}

	err := sendJson(w, &fileList)
	if err != nil {
		return err
	}
	return nil
}

// sendJson - отправляет данные на сервер в формате JSON
func sendJson(w *http.ResponseWriter, files *[]filescanpack.FileInfo) error {
	jsonData, err := json.Marshal(files)
	if err != nil {
		return err
	}
	(*w).Header().Set("Content-Type", "application/json")
	(*w).Write(jsonData)

	return nil
}

func getPort() (string, error) {
	data, err := ioutil.ReadFile("./resources/port.config")
	if err != nil {
		return "", err
	}
	port := strings.TrimSpace(string(data)) // Удаляем пробелы и символы новой строки
	if port == "" {
		return "", errors.New("Ошибка указания порта, скорректируйте файл port.config")
	}
	return port, nil
}
