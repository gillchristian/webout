const output = document.getElementById("output")
const id = location.pathname.replace(/^\//, '')
const socket = new WebSocket(`ws://localhost:8080/ws/${id}`)
let connected = false

socket.onopen = function () {
    connected = true
}

socket.onerror = function (event) {
    if (!connected) {
        output.innerHTML += `Error: failed to connect`
    } else {
        output.innerHTML += `Error: ${JSON.stringify(event, null, 2)}`
    }
};

socket.onmessage = function (e) {
    output.innerHTML += e.data
};
