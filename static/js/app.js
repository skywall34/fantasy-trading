// Portfolio Chart Initialization
function initializePortfolioChart() {
    const chartCanvas = document.getElementById('portfolioChart');
    if (chartCanvas) {
        const ctx = chartCanvas.getContext('2d');

        // Get data from data attributes
        let timestamps = [];
        let equities = [];
        
        try {
            const timestampsAttr = chartCanvas.getAttribute('data-timestamps');
            const equitiesAttr = chartCanvas.getAttribute('data-equities');
            
            if (timestampsAttr && equitiesAttr) {
                timestamps = JSON.parse(timestampsAttr);
                equities = JSON.parse(equitiesAttr);
            }
        } catch (e) {
            console.error('Error parsing portfolio data:', e);
        }

        // If no data available, use mock data for demonstration
        if (timestamps.length === 0 || equities.length === 0) {
            console.error('No data found for chart, using mock data');
            console.info('Data received: ', equities);
            const labels = [];
            let value = 100000;

            for (let i = 30; i >= 0; i--) {
                const date = new Date();
                date.setDate(date.getDate() - i);
                labels.push(date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }));

                // Random walk with upward trend
                value = value * (1 + (Math.random() - 0.45) * 0.03);
                equities.push(value);
            }
        } else {
            // Convert UNIX timestamps to formatted dates
            timestamps = timestamps.map(ts => {
                const date = new Date(ts * 1000); // Convert seconds to milliseconds
                return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
            });

            // Ensure today is included in the chart
            const today = new Date().toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
            if (timestamps.length > 0 && timestamps[timestamps.length - 1] !== today) {
                timestamps.push(today);
                // Use the last equity value for today (since we don't have real-time data)
                equities.push(equities[equities.length - 1]);
            }
        }

        // Destroy existing chart if it exists
        if (chartCanvas.chart) {
            chartCanvas.chart.destroy();
        }

        chartCanvas.chart = new Chart(ctx, {
            type: 'line',
            data: {
                labels: timestamps.length > 0 ? timestamps : [],
                datasets: [{
                    label: 'Portfolio Value',
                    data: equities,
                    borderColor: '#E31B23',
                    backgroundColor: 'rgba(227, 27, 35, 0.1)',
                    fill: true,
                    tension: 0.4,
                    pointRadius: 0,
                    pointHoverRadius: 6,
                    pointHoverBackgroundColor: '#E31B23',
                    pointHoverBorderColor: '#fff',
                    pointHoverBorderWidth: 2,
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                interaction: {
                    intersect: false,
                    mode: 'index',
                },
                plugins: {
                    legend: {
                        display: false
                    },
                    tooltip: {
                        backgroundColor: '#1A1A1A',
                        titleColor: '#fff',
                        bodyColor: '#fff',
                        padding: 12,
                        displayColors: false,
                        callbacks: {
                            label: function(context) {
                                return '$' + context.parsed.y.toLocaleString('en-US', {
                                    minimumFractionDigits: 2,
                                    maximumFractionDigits: 2
                                });
                            }
                        }
                    }
                },
                scales: {
                    x: {
                        grid: {
                            display: false
                        },
                        ticks: {
                            maxTicksLimit: 8,
                            color: '#9CA3AF'
                        }
                    },
                    y: {
                        grid: {
                            color: '#F3F4F6'
                        },
                        ticks: {
                            callback: function(value) {
                                return '$' + (value / 1000).toFixed(0) + 'K';
                            },
                            color: '#9CA3AF'
                        }
                    }
                }
            }
        });
    }
}

// Initialize chart on page load
document.addEventListener('DOMContentLoaded', function() {
    initializePortfolioChart();
    setupTimeframeButtons();
});

// Setup timeframe button event listeners
function setupTimeframeButtons() {
    const buttons = document.querySelectorAll('[data-period]');
    buttons.forEach(button => {
        button.addEventListener('click', function(e) {
            e.preventDefault();
            const period = this.getAttribute('data-period');
            const timeframe = this.getAttribute('data-timeframe');
            fetchPortfolioData(period, timeframe);
        });
    });
}

// Fetch portfolio data for a specific timeframe
function fetchPortfolioData(period, timeframe) {
    // Show loading state
    const chartContainer = document.querySelector('.h-80');
    const originalContent = chartContainer.innerHTML;
    
    fetch(`/api/portfolio/history?period=${period}&timeframe=${timeframe}`)
        .then(response => response.json())
        .then(data => {
            if (data.timestamps && data.equities) {
                // Update canvas data attributes
                const chartCanvas = document.getElementById('portfolioChart');
                chartCanvas.setAttribute('data-timestamps', JSON.stringify(data.timestamps));
                chartCanvas.setAttribute('data-equities', JSON.stringify(data.equities));
                
                // Reinitialize chart with new data
                initializePortfolioChart();
                
                // Update active button state
                updateActiveButton(period);
            }
        })
        .catch(error => {
            console.error('Error fetching portfolio data:', error);
        });
}

// Update active button styling
function updateActiveButton(activePeriod) {
    const buttons = document.querySelectorAll('[data-period]');
    buttons.forEach(button => {
        if (button.getAttribute('data-period') === activePeriod) {
            button.classList.remove('bg-gray-100', 'text-gray-600');
            button.classList.add('bg-eog-red', 'text-white');
        } else {
            button.classList.remove('bg-eog-red', 'text-white');
            button.classList.add('bg-gray-100', 'text-gray-600');
        }
    });
}

// Reinitialize when HTMX swaps content
document.addEventListener('htmx:afterSwap', function(event) {
    if (event.detail.xhr.status === 200) {
        initializePortfolioChart();
        setupTimeframeButtons();
    }
});
