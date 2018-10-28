setInterval(() => {
  console.log('stdout')
}, 10000)

setInterval(() => {
  console.error('stderr')
}, 11000)

setTimeout(() => {
  process.exit(1)
}, 15000)
