package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"unicode"
)

func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func main() {
	//Лока для сохранения файла
	uploadDir := "uploads"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		fmt.Println("Ошибка создания папки: ", err)
		return
	}
	// Редирект для главной страницы
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/upload", http.StatusFound)
	})

	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			err := r.ParseMultipartForm(10 << 20) // 10MB лимит
			if err != nil {
				http.Error(w, "Ошибка загрузки", http.StatusBadRequest)
				return
			}

			files := r.MultipartForm.File["file"] // срез файлов для multiple
			for _, header := range files {
				file, err := header.Open()
				if err != nil {
					http.Error(w, "Ошибка открытия файла", http.StatusBadRequest)
					continue
				}
				defer file.Close()

				// Нормализация имени файла (фикс иероглифов в именах, если были)
				filename := normalizeFilename(header.Filename)

				out, err := os.Create(filepath.Join(uploadDir, filename))
				if err != nil {
					http.Error(w, "Ошибка сохранения", http.StatusInternalServerError)
					continue
				}
				defer out.Close()

				_, err = io.Copy(out, file)
				if err != nil {
					http.Error(w, "Ошибка копирования", http.StatusInternalServerError)
					continue
				}

				fmt.Fprintln(w, "Файл загружен: "+filename)
			}
		} else {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")

			fmt.Fprint(w, `<!DOCTYPE html>
			<html><head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>FileQRTransfer</title>
			<style>
				body { font-family: Arial, sans-serif; padding: 20px; margin: 0; background: #f0f0f0; }
				h1 { text-align: center; }
				form { max-width: 500px; margin: auto; background: white; padding: 30px; border-radius: 15px; box-shadow: 0 4px 20px rgba(0,0,0,0.1); }
				input[type="file"] { width: 100%; padding: 15px; font-size: 18px; border: 2px dashed #ccc; border-radius: 10px; margin-bottom: 20px; box-sizing: border-box; }
				input[type="submit"] { width: 100%; padding: 15px; font-size: 20px; background: #4CAF50; color: white; border: none; border-radius: 10px; cursor: pointer; }
				input[type="submit"]:hover { background: #45a049; }
				@media (max-width: 600px) {
					input { font-size: 20px; padding: 20px; }
				}
			</style>
			</head><body>
			<h1>Загрузка файлов на ПК</h1>
			<form enctype="multipart/form-data" action="/upload" method="post">
				<input type="file" name="file" multiple>
				<input type="submit" value="Загрузить файлы">
			</form>
			</body></html>`)
		}
	})

	localIP := getLocalIP()
	uploadURL := "http://" + localIP + ":8080/upload"
	fmt.Println("FileQRTransfer v1.0 (alpha)")
	fmt.Println("Сервер запущен!")
	fmt.Println("Открой на ПК: http://localhost:8080/upload")
	fmt.Println("С телефона в той же Wi-Fi сети: " + uploadURL)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Ошибка запуска сервера: ", err)
	}
}

// Функция для нормализации имени файла (если нужны)
func normalizeFilename(name string) string {
	var result []rune
	for _, r := range name {
		if unicode.IsGraphic(r) { // оставляем printable символы
			result = append(result, r)
		} else {
			result = append(result, '_')
		}
	}
	return string(result)
}
