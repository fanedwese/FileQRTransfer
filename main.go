package main

import (
	"encoding/base64"
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
    h1 { text-align: center; color: #333; }
    #mainContainer { max-width: 600px; margin: auto; background: white; padding: 30px; border-radius: 15px; box-shadow: 0 4px 20px rgba(0,0,0,0.1); }
    #selectButton { width: 100%; padding: 20px; font-size: 20px; background: #2196F3; color: white; border: none; border-radius: 10px; cursor: pointer; margin-bottom: 20px; }
    #selectButton:hover { background: #1976D2; }
    #fileCount { text-align: center; font-size: 18px; margin: 10px 0; color: #555; }
    #fileList { margin: 20px 0; padding: 10px; background: #f9f9f9; border-radius: 10px; }
    .fileItem { padding: 10px; background: #fff; margin: 5px 0; border-radius: 8px; display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 10px; }
    .removeBtn { background: #f44336; color: white; border: none; padding: 8px 12px; border-radius: 5px; cursor: pointer; flex-shrink: 0; }
    #progressContainer { width: 100%; background: #ddd; border-radius: 10px; margin: 20px 0; display: none; }
    #progressBar { width: 0%; height: 40px; background: #001F3F; text-align: center; line-height: 40px; color: white; border-radius: 10px; font-size: 18px; }
    #status { text-align: center; font-size: 18px; margin-top: 10px; }
    input[type="file"] { display: none; }
    @media (max-width: 600px) {
        #selectButton, .removeBtn { font-size: 18px; padding: 15px; }
    }
</style>
</head><body>
<h1>Загрузка файлов на ПК</h1>
<div id="mainContainer">
    <button id="selectButton">Выбрать файлы</button>
    <p id="fileCount">Файлов: 0</p>
    <div id="fileList"></div>
    <button id="uploadButton" style="width: 100%; padding: 20px; font-size: 20px; background: #3f7e00ff; color: white; border: none; border-radius: 10px; cursor: pointer; margin-top: 20px;">Загрузить файлы</button>
    <div id="progressContainer">
        <div id="progressBar">0%</div>
    </div>
    <div id="status"></div>
</div>
<input type="file" name="file" id="fileInput" multiple>

<script>
    console.log("JS загружен"); // дебаг

    const selectButton = document.getElementById('selectButton');
    const fileInput = document.getElementById('fileInput');
    const fileList = document.getElementById('fileList');
    const fileCount = document.getElementById('fileCount');
    const progressContainer = document.getElementById('progressContainer');
    const progressBar = document.getElementById('progressBar');
    const status = document.getElementById('status');

    let selectedFiles = new DataTransfer();

    selectButton.addEventListener('click', () => fileInput.click());

    function updateFileCount() {
        const count = selectedFiles.files.length;
        fileCount.textContent = 'Файлов: ' + count;
        selectButton.textContent = count === 0 ? 'Выбрать файлы' : 'Добавить ещё';
    }

    function updateFileList() {
        fileList.innerHTML = '<h3>Выбранные файлы:</h3>';
        const files = selectedFiles.files;
        if (files.length === 0) {
            fileList.innerHTML += '<p>Ничего не выбрано</p>';
            return;
        }
	
	const uploadButton = document.getElementById('uploadButton');
uploadButton.addEventListener('click', function() {
    const files = selectedFiles.files;
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
            selectedFiles = new DataTransfer();
            updateFileList();
            updateFileCount();
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

        const ul = document.createElement('ul');
        ul.style.listStyle = 'none';
        ul.style.padding = '0';

        for (let i = 0; i < files.length; i++) {
            const li = document.createElement('li');
            li.className = 'fileItem';
            li.textContent = files[i].name + ' (' + (files[i].size / 1024 / 1024).toFixed(2) + ' MB)';

            const removeBtn = document.createElement('button');
            removeBtn.className = 'removeBtn';
            removeBtn.textContent = 'Удалить';
            removeBtn.onclick = (function(index) {
                return function() {
                    const dt = new DataTransfer();
                    for (let j = 0; j < files.length; j++) {
                        if (j !== index) dt.items.add(files[j]);
                    }
                    selectedFiles = dt;
                    updateFileList();
                    updateFileCount();
                };
            })(i);

            li.appendChild(removeBtn);
            ul.appendChild(li);
        }
        fileList.appendChild(ul);
    }

    fileInput.addEventListener('change', () => {
        const newFiles = fileInput.files;
        for (let file of newFiles) {
            // Проверка на дубли (по имени и размеру)
            let duplicate = false;
            for (let existing of selectedFiles.files) {
                if (existing.name === file.name && existing.size === file.size) {
                    duplicate = true;
                    break;
                }
            }
            if (!duplicate) {
                selectedFiles.items.add(file);
            }
        }
        fileInput.value = '';
        updateFileList();
        updateFileCount();
    });

    // Загрузка
    const form = document.querySelector('#mainContainer'); // привязываем submit к контейнеру, чтобы не было формы
    form.addEventListener('click', function(e) {
        if (e.target.tagName === 'INPUT' && e.target.type === 'submit') { // если клик по submit (но у нас нет submit, но на всякий)
            e.preventDefault();
            // код загрузки тот же
            const files = selectedFiles.files;
            if (files.length === 0) {
                status.textContent = "Выбери файлы, брат!";
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
                    selectedFiles = new DataTransfer();
                    updateFileList();
                    updateFileCount();
                } else {
                    status.textContent = "Ошибка загрузки";
                }
            };

            const formData = new FormData();
            for (let file of files) {
                formData.append('file', file);
            }
            xhr.send(formData);
        }
    });

    const uploadButton = document.getElementById('uploadButton');
    if (uploadButton) {
        uploadButton.addEventListener('click', function() {
            const files = selectedFiles.files;
            if (files.length === 0) {
                status.textContent = "Выбери файлы, брат!";
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
                    selectedFiles = new DataTransfer();
                    updateFileList();
                    updateFileCount();
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
    }

    updateFileList();
    updateFileCount();
</script>
</body></html>`)
		}
	})

	http.HandleFunc("/qr", func(w http.ResponseWriter, r *http.Request) {
		localIP := getLocalIP()
		url := "http://" + localIP + ":8080/upload"

		qr, err := qrcode.New(url, qrcode.Medium)
		if err != nil {
			http.Error(w, "Ошибка генерации QR", http.StatusInternalServerError)
			return
		}

		qrPNG, err := qr.PNG(512)
		if err != nil {
			http.Error(w, "Ошибка PNG", http.StatusInternalServerError)
			return
		}

		qrBase64 := base64.StdEncoding.EncodeToString(qrPNG)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html>
	<html lang="ru">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>FileQRTransfer — by fanedwese</title>
		<style>
			body {
				margin: 0;
				padding: 0;
				height: 100vh;
				background: linear-gradient(135deg, #1e3c72, #2a5298);
				display: flex;
				justify-content: center;
				align-items: center;
				font-family: 'Arial', sans-serif;
				color: white;
				text-align: center;
			}
			.container {
				background: rgba(255, 255, 255, 0.15);
				padding: 40px;
				border-radius: 20px;
				box-shadow: 0 10px 30px rgba(0,0,0,0.3);
				backdrop-filter: blur(10px);
				max-width: 500px;
			}
			h1 {
				margin-bottom: 20px;
				font-size: 28px;
			}
			.qr-frame {
				background: white;
				padding: 20px;
				border-radius: 15px;
				display: inline-block;
				box-shadow: 0 5px 15px rgba(0,0,0,0.2);
			}
			img {
				width: 300px;
				height: 300px;
			}
			p {
				margin-top: 30px;
				font-size: 18px;
			}
			.social {
				margin-top: 40px;
				font-size: 16px;
			}
			.social a {
				color: #a0d8ff;
				text-decoration: none;
				margin: 0 10px;
			}
			.social a:hover {
				text-decoration: underline;
			}
			@media (max-width: 600px) {
				img { width: 250px; height: 250px; }
				h1 { font-size: 24px; }
			}
		</style>
	</head>
	<body>
		<div class="container">
			<h1>FileQRTransfer</h1>
			<div class="qr-frame">
				<img src="data:image/png;base64,`+qrBase64+`" alt="QR-код для загрузки">
			</div>
			<p>Сканируй телефоном и загружай файлы на этот ПК</p>
			<div class="social">
				Автор: fanedwese<br>
				<a href="https://t.me/fanedwese" target="_blank">Telegram</a> • 
				<a href="https://vk.com/fanedwese" target="_blank">VK</a> • 
				<a href="https://github.com/fanedwese" target="_blank">GitHub</a>
			</div>
		</div>
	</body>
	</html>`)
	})

	fmt.Println("QR код готов! Открой в браузере: http://localhost:8080/qr")
	fmt.Println("Поднеси телефон - сканируй и сразу грузи файлы")

	localIP := getLocalIP()
	uploadURL := "http://" + localIP + ":8080/upload"
	fmt.Println("FileQRTransfer v1.2 (alpha)")
	fmt.Println("Last update: 07.01.2026")
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
