alert(1)
fetch('./ui/script.js')
    .then(response => {
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        return response.json();
    })
    .then(data => {
        // Преобразуйте данные в список
        const list = document.getElementById('myList');
        data.forEach(item => {
            const listItem = document.createElement('li');
            listItem.textContent = item.name; // или любое другое поле из JSON
            list.appendChild(listItem);
        });
    })
    .catch(error => {
        console.log('An error occurred:', error);
    });