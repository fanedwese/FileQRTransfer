package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"unicode"

	"os/exec"

	"github.com/skip2/go-qrcode"
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
			#uploadForm { max-width: 500px; margin: auto; background: white; padding: 30px; border-radius: 15px; box-shadow: 0 4px 20px rgba(0,0,0,0.1); }
			input[type="file"] { width: 100%; padding: 15px; font-size: 18px; border: 2px dashed #ccc; border-radius: 10px; margin-bottom: 20px; box-sizing: border-box; }
			input[type="submit"] { width: 100%; padding: 15px; font-size: 20px; background: #4CAF50; color: white; border: none; border-radius: 10px; cursor: pointer; }
			#progressContainer { width: 100%; background: #ddd; border-radius: 10px; margin: 20px 0; display: none; }
			#progressBar { width: 0%; height: 30px; background: #00414b; text-align: center; line-height: 30px; color: white; border-radius: 10px; }
			#status { text-align: center; font-size: 18px; }
		</style>
		</head><body>
		<h1>Загрузка файлов на ПК</h1>
		<div id="uploadForm">
			<form id="form">
				<input type="file" name="file" id="fileInput" multiple>
				<input type="submit" value="Загрузить файлы">
			</form>
			<div id="progressContainer">
				<div id="progressBar">0%</div>
			</div>
			<div id="status"></div>
		</div>

	<script>
		const form = document.getElementById('form');
		const fileInput = document.getElementById('fileInput');
		const status = document.getElementById('status');
		const progressContainer = document.getElementById('progressContainer');
		const progressBar = document.getElementById('progressBar');

		// Контейнер для списка файлов
		const fileList = document.createElement('div');
		fileList.id = 'fileList';
		fileList.style.marginTop = '20px';
		fileList.style.padding = '10px';
		fileList.style.background = '#f9f9f9';
		fileList.style.borderRadius = '10px';
		fileInput.parentNode.insertBefore(fileList, fileInput.nextSibling);

		// Обновление списка выбранных файлов
		function updateFileList() {
			fileList.innerHTML = '<h3>Выбранные файлы:</h3>';
			const files = fileInput.files;
			if (files.length === 0) {
				fileList.innerHTML += '<p>Ничего не выбрано</p>';
				return;
			}

			const ul = document.createElement('ul');
			ul.style.listStyle = 'none';
			ul.style.padding = '0';

			for (let i = 0; i < files.length; i++) {
				const li = document.createElement('li');
				li.style.padding = '10px';
				li.style.background = '#fff';
				li.style.margin = '5px 0';
				li.style.borderRadius = '8px';
				li.style.display = 'flex';
				li.style.justifyContent = 'space-between';
				li.style.alignItems = 'center';
				li.style.wordBreak = 'break-all';

				li.textContent = files[i].name + ' (' + (files[i].size / 1024 / 1024).toFixed(2) + ' MB)';

				const removeBtn = document.createElement('button');
				removeBtn.textContent = 'Удалить';
				removeBtn.style.background = '#f44336';
				removeBtn.style.color = 'white';
				removeBtn.style.border = 'none';
				removeBtn.style.padding = '5px 10px';
				removeBtn.style.borderRadius = '5px';
				removeBtn.style.cursor = 'pointer';
				removeBtn.onclick = (function(index) {
					return function() {
						const dt = new DataTransfer();
						for (let j = 0; j < files.length; j++) {
							if (j !== index) dt.items.add(files[j]);
						}
						fileInput.files = dt.files;
						updateFileList();
					}
				})(i);

				li.appendChild(removeBtn);
				ul.appendChild(li);
			}
			fileList.appendChild(ul);
		}

		// Обновляем список при выборе файлов
		fileInput.addEventListener('change', updateFileList);

		// Загрузка с прогрессом (твой старый код, но с небольшим фиксом)
		form.addEventListener('submit', function(e) {
			e.preventDefault();
			const files = fileInput.files;
			if (files.length === 0) {
				status.textContent = "Выбери файлы!";
				return;
			}

			const xhr = new XMLHttpRequest();
			xhr.open('POST', '/upload', true);

			xhr.upload.onprogress = function(event) {
				if (event.lengthComputable) {
					const percent = (event.loaded / event.total) * 100;
					progressBar.style.width = percent + '%';
					progressBar.textContent = Math.round(percent) + '%';
					progressContainer.style.display = 'block';
				}
			};

			xhr.onload = function() {
				if (xhr.status === 200) {
					status.textContent = "Все файлы загружены!";
					progressBar.style.width = '100%';
					progressBar.textContent = '100%';
					fileInput.value = ''; // очищаем выбор
					updateFileList(); // очищаем список
				} else {
					status.textContent = "Ошибка загрузки";
				}
			};

			const formData = new FormData();
			for (let file of files) {
				formData.append('file', file);
			}

			xhr.send(formData);
		});

		// Инициализация списка
		updateFileList();
	</script>
		</body></html>`)
		}
	})

	http.HandleFunc("/qr", func(w http.ResponseWriter, r *http.Request) {
		localIP := getLocalIP()
		url := "http://" + localIP + ":8080/upload"

		qr, err := qrcode.New(url, qrcode.Medium)
		if err != nil {
			http.Error(w, "Не смог сделать QR код", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		qr.Write(512, w)
	})

	fmt.Println("QR код готов! Открой в браузере: http://localhost:8080/qr")
	fmt.Println("Поднеси телефон - сканируй и сразу грузи файлы")

	localIP := getLocalIP()
	uploadURL := "http://" + localIP + ":8080/upload"
	fmt.Println("FileQRTransfer v1.1 (alpha)")
	fmt.Println("Last update: 04.01.2026")
	fmt.Println("Сервер запущен!")
	fmt.Println("Открой на ПК: http://localhost:8080/upload")
	fmt.Println("С телефона в той же Wi-Fi сети: " + uploadURL)
	go func() {
		url := "http://localhost:8080/qr"
		exec.Command("cmd", "/c", "start", "", url).Start()
	}()
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
