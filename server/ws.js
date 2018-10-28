const output = document.getElementById('output')

const socket = new WebSocket(wsUrl())

let connected = false

socket.onopen = function () {
    connected = true
}

socket.onerror = function (event) {
    if (!connected) {
        output.innerHTML += 'Error: failed to connect'
    } else {
        output.innerHTML += `Error: ${JSON.stringify(event, null, 2)}`
    }
};

socket.onmessage = function (e) {
    output.innerHTML += e.data
};

function wsUrl() {
    const protocol = location.protocol.match(/https/) ? 'wss' : 'ws'
    const pathname = location.host.match(/local/) ? '' : '/netpipe'
    const id = location.pathname.replace(/\//g, '').replace('netpipe', '')

    return `${protocol}://${location.host}${pathname}/ws/${id}`
}
