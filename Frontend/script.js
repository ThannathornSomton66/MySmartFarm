// const relayMap = {};
async function loadData() {
    try {
      const [dataRes, intervalRes, relayRes] = await Promise.all([
        fetch("http://127.0.0.1:3000/api/v1/data"),
        fetch("http://127.0.0.1:3000/api/v1/intervals"),
        fetch("http://127.0.0.1:3000/api/v1/relays")
      ]);
  
      const data = await dataRes.json();
      const intervals = await intervalRes.json();
      const relays = await relayRes.json();
  
      const intervalMap = {};
      intervals.forEach(i => {
        intervalMap[i.DeviceID] = i.IntervalSeconds;
      });
  
        const relayMap = {};
        relays.forEach(r => {
        relayMap[r.device_id] = r.ip;
      });
  
      const container = document.getElementById("sensor-grid");
      container.innerHTML = "";
  
      const latestByDevice = {};
      for (const d of data) {
        const existing = latestByDevice[d.DeviceID];
        if (!existing || new Date(d.Timestamp) > new Date(existing.Timestamp)) {
          latestByDevice[d.DeviceID] = d;
        }
      }
  
      Object.values(latestByDevice).forEach(d => {
        const currentInterval = intervalMap[d.DeviceID] || 300;
  
        const card = document.createElement("div");
        card.className = "card";
  
        card.innerHTML = `
          <div class="device-id">${d.DeviceID}</div>
          <div>🌡️ Temp: ${d.Temperature}°C</div>
          <div>💧 Humidity: ${d.Humidity}%</div>
          <div>🪴 Soil: ${d.Soil}%</div>
          <div class="timestamp">Last updated: ${new Date(d.Timestamp).toLocaleString()}</div>
          <label>
            ⏱️ Interval:
            <select data-device="${d.DeviceID}" class="interval-select">
              <option value="60" ${currentInterval == 60 ? "selected" : ""}>1 min</option>
              <option value="300" ${currentInterval == 300 ? "selected" : ""}>5 min</option>
              <option value="900" ${currentInterval == 900 ? "selected" : ""}>15 min</option>
              <option value="1800" ${currentInterval == 1800 ? "selected" : ""}>30 min</option>
              <option value="3600" ${currentInterval == 3600 ? "selected" : ""}>1 hour</option>
            </select>
          </label>
        <div>
        <button onclick="controlRelay('${d.DeviceID}', 'on')">Relay ON</button>
        <button onclick="controlRelay('${d.DeviceID}', 'off')">Relay OFF</button>
        </div>
        `;
  
        container.appendChild(card);
      });
    } catch (err) {
      console.error("Failed to load sensor data or intervals:", err);
    }
  }
  
  async function controlRelay(deviceID, action) {
    try {
      const res = await fetch(`http://127.0.0.1:3000/api/v1/relay/${deviceID}/${action}`, {
        method: "POST"
      });
  
      if (res.ok) {
        console.log(`Relay ${action} command sent to ${deviceID}`);
      } else {
        const errMsg = await res.text();
        alert("Failed to control relay: " + errMsg);
      }
    } catch (err) {
      console.error("Relay error:", err);
      alert("Relay request failed.");
    }
  }
  
  document.addEventListener("change", async (e) => {
    if (e.target.classList.contains("interval-select")) {
      const deviceID = e.target.getAttribute("data-device");
      const intervalSeconds = parseInt(e.target.value);
  
      try {
        const res = await fetch("http://127.0.0.1:3000/api/v1/interval", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ deviceID, intervalSeconds })
        });
  
        if (res.ok) {
          alert(`Interval updated to ${intervalSeconds} seconds for ${deviceID}`);
        } else {
          alert("Failed to update interval");
        }
      } catch (err) {
        console.error("Error setting interval:", err);
      }
    }
  });
  
  loadData();
  setInterval(loadData, 60000); // Refresh every 60 seconds
  
  // ======== Chart for Temperature & Humidity ========
  
  function getTimeAxisOptions() {
    const val = document.getElementById("xAxisInterval").value;
    if (!val) return {}; // Auto-calculate
    const [unit, step] = val.split("-");
    return {
      unit,
      stepSize: parseInt(step),
      displayFormats: { [unit]: "HH:mm" }
    };
  }
  
  let tempChartInstance = null;
  
  async function drawTempChart() {
    const res = await fetch('http://localhost:3000/api/v1/data/device/sensor-001');
    const data = await res.json();
  
    const labels = data.map(d => new Date(d.Timestamp));
    const temps = data.map(d => d.Temperature);
    const hums = data.map(d => d.Humidity);
  
    if (tempChartInstance) {
      tempChartInstance.destroy();
    }
  
    const ctx = document.getElementById('tempChart').getContext('2d');
    tempChartInstance = new Chart(ctx, {
      type: 'line',
      data: {
        labels,
        datasets: [
          {
            label: '🌡️ Temperature (°C)',
            data: temps,
            borderColor: 'rgba(255, 99, 132, 1)',
            backgroundColor: 'rgba(255, 99, 132, 0.2)',
            yAxisID: 'y',
            tension: 0.3,
            pointRadius: 3
          },
          {
            label: '💧 Humidity (%)',
            data: hums,
            borderColor: 'rgba(54, 162, 235, 1)',
            backgroundColor: 'rgba(54, 162, 235, 0.2)',
            yAxisID: 'y1',
            tension: 0.3,
            pointRadius: 3
          }
        ]
      },
      options: {
        responsive: true,
        interaction: {
          mode: 'index',
          intersect: false
        },
        scales: {
          x: {
            type: 'time',
            time: getTimeAxisOptions(),
            title: {
              display: true,
              text: 'Time'
            }
          },
          y: {
            type: 'linear',
            position: 'left',
            title: {
              display: true,
              text: 'Temperature (°C)'
            }
          },
          y1: {
            type: 'linear',
            position: 'right',
            title: {
              display: true,
              text: 'Humidity (%)'
            },
            grid: {
              drawOnChartArea: false
            }
          }
        }
      }
    });
  }
  
  document.getElementById("xAxisInterval").addEventListener("change", drawTempChart);
  drawTempChart();
  