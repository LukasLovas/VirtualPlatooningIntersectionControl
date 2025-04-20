document.addEventListener('DOMContentLoaded', () => {
    let ws;
    let liveChart;
    let comparisonCharts = {};
    let metricsHistory = {
        timestamps: [],
        vehicleCount: [],
        avgSpeed: [],
        throughput: [],
        avgWaitTime: []
    };
    
    let sumoFiles = [];
    let customFiles = [];
    let lastWebSocketMessage = Date.now();
    let reconnectTimeout;
    let simulationStarted = false;
    
    initWebSocket();
    initCharts();
    loadStatisticsFiles();
    setupEventListeners();
    
    function initWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws`;
        
        if (ws) {
            try {
                ws.close();
            } catch (e) {}
        }
        
        ws = new WebSocket(wsUrl);
        
        ws.onopen = () => {
            console.log('webSocket conn ');
            
            if (reconnectTimeout) {
                clearTimeout(reconnectTimeout);
                reconnectTimeout = null;
            }
        };
        
        ws.onclose = () => {
            console.log('webSocket connection closed');
            
            if (!reconnectTimeout) {
                reconnectTimeout = setTimeout(() => {
                    initWebSocket();
                    reconnectTimeout = null;
                }, 2000);
            }
        };
        
        ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
        
        ws.onmessage = (event) => {
            lastWebSocketMessage = Date.now();
            try {
                const data = JSON.parse(event.data);
                updateMetricsDisplay(data);
                updateLiveChart(data);
                
                if (!simulationStarted && data.time_step > 0) {
                    simulationStarted = true;
                    
                    if (data.using_custom_algo !== undefined) {
                        updateAlgorithmDisplay(data.using_custom_algo);
                    }
                }
            } catch (e) {
                console.error('Error processing WebSocket message:', e);
            }
        };
    }
    
    function updateAlgorithmDisplay(isCustom) {
        const algoName = isCustom ? 'Custom' : 'SUMO';
        const algoClass = isCustom ? 'custom' : 'sumo';
        
        document.getElementById('algorithm').textContent = algoName;
        document.getElementById('algorithm-select').value = isCustom ? 'custom' : 'sumo';
        
        const badge = document.getElementById('current-algorithm-badge');
        badge.textContent = algoName;
        badge.className = `algo-badge ${algoClass}`;
    }
    
    setInterval(() => {
        const now = Date.now();
        if (now - lastWebSocketMessage > 5000) {
            console.log('webSocket appears inactive, lol, attempting to reconnect');
            initWebSocket();
        }
    }, 5000);
    
    function initCharts() {
        const liveCtx = document.getElementById('live-chart').getContext('2d');
        liveChart = new Chart(liveCtx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [
                    {
                        label: 'Vehicles',
                        data: [],
                        borderColor: 'rgb(52, 152, 219)',
                        backgroundColor: 'rgba(52, 152, 219, 0.1)',
                        tension: 0.3,
                        borderWidth: 2,
                        fill: true
                    },
                    {
                        label: 'Avg Speed (m/s)',
                        data: [],
                        borderColor: 'rgb(46, 204, 113)',
                        backgroundColor: 'rgba(46, 204, 113, 0.1)',
                        tension: 0.3,
                        borderWidth: 2,
                        fill: true
                    }
                ]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    x: {
                        ticks: {
                            maxTicksLimit: 10
                        }
                    }
                },
                plugins: {
                    legend: {
                        position: 'bottom'
                    }
                },
                animation: {
                    duration: 0
                }
            }
        });
        
        const chartIds = [
            { id: 'throughput-chart', label: 'Total Throughput' },
            { id: 'wait-times-chart', label: 'Average Wait Time (s)' },
            { id: 'speed-chart', label: 'Average Speed (m/s)' },
            { id: 'platoons-chart', label: 'Platoon Size' }
        ];
        
        chartIds.forEach(({id, label}) => {
            const ctx = document.getElementById(id).getContext('2d');
            comparisonCharts[id] = new Chart(ctx, {
                type: 'line',
                data: {
                    labels: [],
                    datasets: [
                        {
                            label: 'SUMO Algorithm',
                            data: [],
                            borderColor: 'rgb(231, 76, 60)',
                            backgroundColor: 'rgba(231, 76, 60, 0.1)',
                            tension: 0.3,
                            borderWidth: 2
                        },
                        {
                            label: 'Custom Algorithm',
                            data: [],
                            borderColor: 'rgb(52, 152, 219)',
                            backgroundColor: 'rgba(52, 152, 219, 0.1)',
                            tension: 0.3,
                            borderWidth: 2
                        }
                    ]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    scales: {
                        y: {
                            beginAtZero: true,
                            title: {
                                display: true,
                                text: label
                            }
                        },
                        x: {
                            title: {
                                display: true,
                                text: 'Time Step'
                            },
                            ticks: {
                                maxTicksLimit: 10
                            }
                        }
                    },
                    plugins: {
                        legend: {
                            position: 'bottom'
                        }
                    },
                    animation: {
                        duration: 0
                    }
                }
            });
        });
    }
    
    function updateMetricsDisplay(data) {
        document.getElementById('time-step').textContent = data.time_step || 0;
        document.getElementById('algorithm').textContent = data.using_custom_algo ? 'Custom' : 'SUMO';
        document.getElementById('vehicle-count').textContent = data.vehicle_count || 0;
        document.getElementById('platoon-count').textContent = data.platoon_count || 0;
        document.getElementById('average-speed').textContent = `${(data.average_speed || 0).toFixed(1)} m/s`;
        document.getElementById('throughput').textContent = data.total_throughput || 0;
        document.getElementById('intersection-count').textContent = data.intersection_count || 0;
        document.getElementById('avg-wait-time').textContent = `${(data.average_wait_time || 0).toFixed(1)} s`;
        document.getElementById('queue-size').textContent = data.intersection_queue_size || 0;
        document.getElementById('traffic-density').textContent = `${(data.traffic_density || 0).toFixed(1)}%`;
    }
    
    function updateLiveChart(data) {
        const maxDataPoints = 200;
        
        const timeStep = data.time_step || 0;
        
        if (metricsHistory.timestamps.length > 0 && 
            timeStep <= metricsHistory.timestamps[metricsHistory.timestamps.length - 1]) {
            return;  
        }
        
        metricsHistory.timestamps.push(timeStep);
        metricsHistory.vehicleCount.push(data.vehicle_count || 0);
        metricsHistory.avgSpeed.push(data.average_speed || 0);
        metricsHistory.throughput.push(data.total_throughput || 0);
        metricsHistory.avgWaitTime.push(data.average_wait_time || 0);
        
        while (metricsHistory.timestamps.length > maxDataPoints) {
            metricsHistory.timestamps.shift();
            metricsHistory.vehicleCount.shift();
            metricsHistory.avgSpeed.shift();
            metricsHistory.throughput.shift();
            metricsHistory.avgWaitTime.shift();
        }
        
        liveChart.data.labels = [...metricsHistory.timestamps];
        liveChart.data.datasets[0].data = [...metricsHistory.vehicleCount];
        liveChart.data.datasets[1].data = [...metricsHistory.avgSpeed];
        liveChart.update();
    }
    
    function loadStatisticsFiles() {
        fetch('/api/stats')
            .then(response => response.json())
            .then(data => {
                const sumoFileList = document.getElementById('sumo-file-list');
                const customFileList = document.getElementById('custom-file-list');
                
                sumoFileList.innerHTML = '';
                customFileList.innerHTML = '';
                
                sumoFiles = [];
                customFiles = [];
                
                if (data.files && data.files.length > 0) {
                    data.files.forEach(file => {
                        const fileItem = document.createElement('div');
                        fileItem.className = 'file-item';
                        fileItem.dataset.path = file.path;
                        fileItem.dataset.algo = file.algo || (file.name.toLowerCase().includes('sumo') ? 'sumo' : 'custom');
                        
                        const fileName = document.createElement('div');
                        fileName.className = 'file-name';
                        fileName.textContent = file.name;
                        
                        const fileInfo = document.createElement('div');
                        fileInfo.className = 'file-info';
                        fileInfo.textContent = `${file.size} • ${file.time}`;
                        
                        fileItem.appendChild(fileName);
                        fileItem.appendChild(fileInfo);
                        
                        if (fileItem.dataset.algo === 'sumo') {
                            sumoFileList.appendChild(fileItem);
                            sumoFiles.push(file.path);
                        } else {
                            customFileList.appendChild(fileItem);
                            customFiles.push(file.path);
                        }
                        
                        fileItem.addEventListener('click', () => {
                            if (fileItem.dataset.algo === 'sumo') {
                                loadAndCompareFiles(sumoFiles, customFiles);
                            } else {
                                loadAndCompareFiles(sumoFiles, customFiles);
                            }
                        });
                    });
                    
                    if (sumoFiles.length === 0) {
                        sumoFileList.innerHTML = '<p>No SUMO statistics files found</p>';
                    } else {
                        loadAndCompareFiles(sumoFiles, customFiles);
                    }
                    
                    if (customFiles.length === 0) {
                        customFileList.innerHTML = '<p>No Custom statistics files found</p>';
                    }
                } else {
                    sumoFileList.innerHTML = '<p>No statistics files found</p>';
                    customFileList.innerHTML = '<p>No statistics files found</p>';
                }
                
                setupTabNavigation();
            })
            .catch(error => {
                console.error('Error loading statistics:', error);
                const sumoFileList = document.getElementById('sumo-file-list');
                const customFileList = document.getElementById('custom-file-list');
                sumoFileList.innerHTML = '<p>Error loading statistics files</p>';
                customFileList.innerHTML = '<p>Error loading statistics files</p>';
            });
    }
    
    function loadAndCompareFiles(sumoFiles, customFiles) {
        let sumoData = [];
        let customData = [];
        let loadedCount = 0;
        let totalToLoad = sumoFiles.length + customFiles.length;
        
        if (totalToLoad === 0) {
            return;
        }
        
        sumoFiles.forEach(filePath => {
            loadCsvFileData(filePath, 'sumo', (data) => {
                if (data) {
                    sumoData.push(data);
                }
                loadedCount++;
                if (loadedCount === totalToLoad) {
                    processAndDisplayComparison(sumoData, customData);
                }
            });
        });
        
        customFiles.forEach(filePath => {
            loadCsvFileData(filePath, 'custom', (data) => {
                if (data) {
                    customData.push(data);
                }
                loadedCount++;
                if (loadedCount === totalToLoad) {
                    processAndDisplayComparison(sumoData, customData);
                }
            });
        });
    }
    
    function loadCsvFileData(filePath, algoType, callback) {
        console.log(`lding CSV data from: ${filePath}`);
        fetch(`/api/csv-data?file=${encodeURIComponent(filePath)}`)
            .then(response => {
                if (!response.ok) {
                    throw new Error(`HTTP error ${response.status}`);
                }
                return response.json();
            })
            .then(data => {
                console.log(`CSV data loaded successfully for ${algoType} algorithm:`, filePath);
                callback({ ...data, algoType: algoType });
            })
            .catch(error => {
                console.error('err loading CSV data:', error, filePath);
                callback(null);
            });
    }
    
    function processAndDisplayComparison(sumoDataArray, customDataArray) {
        if ((!sumoDataArray || sumoDataArray.length === 0) && (!customDataArray || customDataArray.length === 0)) {
            console.warn('No data available for comparison');
            return;
        }
        
        const aggregatedSumoData = aggregateDataFromFiles(sumoDataArray);
        const aggregatedCustomData = aggregateDataFromFiles(customDataArray);
        
        updateComparisonCharts(aggregatedSumoData, aggregatedCustomData);
    }
    
    function aggregateDataFromFiles(dataArray) {
        if (!dataArray || dataArray.length === 0) {
            return null;
        }
        
        const allTimeSteps = [];
        const throughputValues = {};
        const waitTimeValues = {};
        const speedValues = {};
        const platoonSizeValues = {};
        
        dataArray.forEach(data => {
            if (!data || !data.rows) return;
            
            data.rows.forEach(row => {
                const timeStep = parseInt(row.TimeStep || 0);
                if (!allTimeSteps.includes(timeStep)) {
                    allTimeSteps.push(timeStep);
                }
                
                if (!throughputValues[timeStep]) throughputValues[timeStep] = [];
                if (!waitTimeValues[timeStep]) waitTimeValues[timeStep] = [];
                if (!speedValues[timeStep]) speedValues[timeStep] = [];
                if (!platoonSizeValues[timeStep]) platoonSizeValues[timeStep] = [];
                
                const throughput = row.TotalThroughput ? parseInt(row.TotalThroughput) : 
                                   (row.ThroughputCount ? parseInt(row.ThroughputCount) : 0);
                const waitTime = parseFloat(row.AverageWaitTime || 0);
                const speed = parseFloat(row.AverageSpeed || 0);
                const platoonSize = parseFloat(row.AveragePlatoonSize || 0);
                
                throughputValues[timeStep].push(throughput);
                waitTimeValues[timeStep].push(waitTime);
                speedValues[timeStep].push(speed);
                platoonSizeValues[timeStep].push(platoonSize);
            });
        });
        
        allTimeSteps.sort((a, b) => a - b);
        
        const timeSteps = allTimeSteps;
        const throughputData = [];
        const waitTimeData = [];
        const speedData = [];
        const platoonSizeData = [];
        
        timeSteps.forEach(step => {
            if (throughputValues[step] && throughputValues[step].length > 0) {
                const avg = throughputValues[step].reduce((a, b) => a + b, 0) / throughputValues[step].length;
                throughputData.push(avg);
            } else {
                throughputData.push(null);
            }
            
            if (waitTimeValues[step] && waitTimeValues[step].length > 0) {
                const avg = waitTimeValues[step].reduce((a, b) => a + b, 0) / waitTimeValues[step].length;
                waitTimeData.push(avg);
            } else {
                waitTimeData.push(null);
            }
            
            if (speedValues[step] && speedValues[step].length > 0) {
                const avg = speedValues[step].reduce((a, b) => a + b, 0) / speedValues[step].length;
                speedData.push(avg);
            } else {
                speedData.push(null);
            }
            
            if (platoonSizeValues[step] && platoonSizeValues[step].length > 0) {
                const avg = platoonSizeValues[step].reduce((a, b) => a + b, 0) / platoonSizeValues[step].length;
                platoonSizeData.push(avg);
            } else {
                platoonSizeData.push(null);
            }
        });
        
        return {
            timeSteps,
            throughputData,
            waitTimeData,
            speedData,
            platoonSizeData
        };
    }
    
    function updateComparisonCharts(sumoData, customData) {
        const charts = {
            'throughput-chart': { dataKey: 'throughputData', label: 'Total Throughput' },
            'wait-times-chart': { dataKey: 'waitTimeData', label: 'Average Wait Time (s)' },
            'speed-chart': { dataKey: 'speedData', label: 'Average Speed (m/s)' },
            'platoons-chart': { dataKey: 'platoonSizeData', label: 'Average Platoon Size' }
        };
        
        Object.entries(charts).forEach(([chartId, config]) => {
            const chart = comparisonCharts[chartId];
            if (!chart) return;
            
            const allTimeSteps = [];
            
            if (sumoData && sumoData.timeSteps) {
                sumoData.timeSteps.forEach(step => {
                    if (!allTimeSteps.includes(step)) allTimeSteps.push(step);
                });
            }
            
            if (customData && customData.timeSteps) {
                customData.timeSteps.forEach(step => {
                    if (!allTimeSteps.includes(step)) allTimeSteps.push(step);
                });
            }
            
            allTimeSteps.sort((a, b) => a - b);
            
            chart.data.labels = allTimeSteps;
            
            if (sumoData) {
                chart.data.datasets[0].data = mapDataToTimeSteps(
                    sumoData.timeSteps,
                    sumoData[config.dataKey],
                    allTimeSteps
                );
            } else {
                chart.data.datasets[0].data = [];
            }
            
            if (customData) {
                chart.data.datasets[1].data = mapDataToTimeSteps(
                    customData.timeSteps,
                    customData[config.dataKey],
                    allTimeSteps
                );
            } else {
                chart.data.datasets[1].data = [];
            }
            
            chart.update();
        });
    }
    
    function mapDataToTimeSteps(originalTimeSteps, originalData, targetTimeSteps) {
        const result = [];
        
        targetTimeSteps.forEach(targetStep => {
            const index = originalTimeSteps.indexOf(targetStep);
            if (index !== -1) {
                result.push(originalData[index]);
            } else {
                result.push(null);
            }
        });
        
        return result;
    }
    
    function setupEventListeners() {
        document.getElementById('btn-change-algo').addEventListener('click', () => {
            const algorithm = document.getElementById('algorithm-select').value;
            const currentAlgo = document.getElementById('algorithm').textContent.toLowerCase();
            
            if (currentAlgo === algorithm) {
                alert(`Already using ${algorithm} algorithm`);
                return;
            }
            
            sendControlCommand('change_algo', { algorithm });
            
            setTimeout(() => {
                loadStatisticsFiles();
            }, 2000);
        });
        
        setupTabNavigation();
    }
    
    function setupTabNavigation() {
        const tabs = document.querySelectorAll('.tab-btn');
        tabs.forEach(tab => {
            tab.addEventListener('click', () => {
                const targetTab = tab.dataset.tab;
                
                document.querySelectorAll('.tab-btn').forEach(t => {
                    t.classList.remove('active');
                });
                
                document.querySelectorAll('.tab-pane').forEach(p => {
                    p.classList.remove('active');
                });
                
                tab.classList.add('active');
                document.getElementById(`${targetTab}-tab`).classList.add('active');
                
                setTimeout(() => {
                    Object.values(comparisonCharts).forEach(chart => {
                        chart.update();
                    });
                }, 10);
            });
        });
    }
    
    function sendControlCommand(action, params = {}) {
        console.log(`sending control command: ${action}`, params);
        
        const url = new URL('/api/control', window.location.origin);
        url.searchParams.append('action', action);
        
        Object.entries(params).forEach(([key, value]) => {
            url.searchParams.append(key, value);
        });
        
        fetch(url, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded',
            },
        })
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP error ${response.status}`);
            }
            return response.json();
        })
        .then(data => {
            console.log('control command response:', data);
            
            if (action === 'change_algo') {
                updateAlgorithmDisplay(params.algorithm === 'custom');
                
                metricsHistory = {
                    timestamps: [],
                    vehicleCount: [],
                    avgSpeed: [],
                    throughput: [],
                    avgWaitTime: []
                };
                
                liveChart.data.labels = [];
                liveChart.data.datasets[0].data = [];
                liveChart.data.datasets[1].data = [];
                liveChart.update();
            }
            
            console.log(data.message || 'command executed successfully');
        })
        .catch(error => {
            console.error('Error sending control command:', error);
            alert(`Failed to execute ${action} command: ${error.message}`);
        });
    }
});