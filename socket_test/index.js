// Browser console da ishlating
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = function(event) {
    console.log('✅ WebSocket ulandi');
    ws.send('Salom WebSocket!');
};

ws.onmessage = function(event) {
    console.log('📨 Xabar keldi:', event.data);
    const message = JSON.parse(event.data);
    console.log('Xabar tafsilotlari:', message);
};

ws.onclose = function(event) {
    console.log('❌ WebSocket uzildi');
};

ws.onerror = function(error) {
    console.error('💥 Xatolik:', error);
};