package main

import (
	"bufio"
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

	// Создаем контекст, который может быть отменен
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Устанавливаем обработчик для корневого пути
	http.HandleFunc("/files", filesHandler)
	http.HandleFunc("/", htmlHandler)
	http.Handle("/files/ui/", http.StripPrefix("/files/ui/", http.FileServer(http.Dir("./ui"))))
	port, err := getPort()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Запускаем HTTP-сервер на порту 8080
	fmt.Println("Запуск сервера на http://localhost:" + port)

	srv := &http.Server{Addr: ":" + port}

	// Запускаем сервер в отдельном горутине
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Ошибка при запуске сервера: %v\n", err)
			os.Exit(1)
		}
	}()

	// Канал для сигналов прерывания (CTRL+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Ожидаем сигнал прерывания
	<-sigChan

	// Завершаем работу сервера
	fmt.Println("\nЗавершаем работу сервера...")
	srv.Shutdown(ctx)

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
func htmlHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	htmlFile, err := os.Open("./ui/index_page.html")
	if err != nil {
		http.Error(w, "Не удалось открыть файл!", http.StatusInternalServerError)
		return
	}
	defer htmlFile.Close()

	scanner := bufio.NewScanner(htmlFile)
	for scanner.Scan() {
		w.Write(scanner.Bytes())
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
	fileList, err := filescanpack.GetDirectoryContents(directorySource)

	if err != nil {
		return err
	}
	filescanpack.MustSortDirectoryContents(fileList, sortBy)
	err = sendJsonViewOnServer(w, fileList)
	if err != nil {
		return err
	}
	return nil
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
