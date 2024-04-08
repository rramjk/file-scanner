document.addEventListener('DOMContentLoaded',  async function() {
    let url = new URL(window.location.href);
    let params = new URLSearchParams(url.search)
    let root = params.get('root')
    let sort = params.get('sort')
    if (root == null) root = ""
    if (sort == null) sort = ""
    url.href = url.protocol+"//"+url.hostname+":"+url.port+"/files"

    let paramsFiles = new URLSearchParams();
    paramsFiles.append('root', root);
    paramsFiles.append('sort', sort);
    url.search = paramsFiles.toString();

    console.log(url.href)
    try {
        const response = await fetch(url.href, {
            method: 'GET',
        });
        if (!response.ok) {
            throw new Error(`Ошибка HTTP: ${response.status}`)
        }
        const dataInfo = await response.json();
        const data = dataInfo.FilesInfos
        let time = document.querySelector(".time")
        time.innerText = formatTime(dataInfo.Time)
        let container = document.querySelector(".loaded-views");
        for (i=0; i<data.length; i++) {
            let fileInfos = document.createElement("li");
            fileInfos.className = "fileInfos";
    
            let fileInfo = document.createElement("div");
            fileInfo.className = "li";
    
            let type = document.createElement("article");
            type.className = "type";
            type.id = "info"
            type.innerHTML = data[i].IsDir;
    
            let name = document.createElement("article");
            name.className = "name";
            name.id = "info"
            name.innerHTML = data[i].Name;
    
            let size = document.createElement("article");
            size.className = "size";
            size.id = "info"
            size.innerHTML = data[i].Size;
    
            fileInfo.appendChild(type);
            fileInfo.appendChild(name);
            fileInfo.appendChild(size);
    
            fileInfos.appendChild(fileInfo);
    
            container.appendChild(fileInfos);
            fileInfo.addEventListener('click', function() {
                let url = new URL(window.location.href);
                let root = url.searchParams.get("root");
                if (root != null && root != "") {
                    let arrRoot = root.split("/");
                    if (arrRoot[arrRoot.length-1] == "") arrRoot.pop();
                    if (type.innerText == "Dir" && fileInfo.className != "root") {
                        root += "/" + name.innerText + "/";
                        url.searchParams.set('root', root);
                        window.location.href = url;
                    }
                } else if (root == null) {
                    url.searchParams.append("root", "/home");
                    window.location.href = url;
                } else if (root == "") {
                    root = "/home";
                    url.searchParams.set('root', root);
                    window.location.href = url;
                }
            });
        }

    } catch (error) {
        console.log("Ошибка при отправке запроса:", error)
    }

})
