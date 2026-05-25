const API_BASE = '';

let jwtToken = localStorage.getItem('comic_jwt') || null;
let currentView = 'search';

let currentDownloadUrl = '';
let currentComicId = '';

// DOM Elements
const loginBtn = document.getElementById('loginBtn');
const logoutBtn = document.getElementById('logoutBtn');
const goToAdminBtn = document.getElementById('goToAdminBtn');
const goToSearchBtn = document.getElementById('goToSearchBtn');

const mainView = document.getElementById('mainView');
const adminView = document.getElementById('adminView');

const loginModal = document.getElementById('loginModal');
const closeModal = document.getElementById('closeModal');
const submitLoginBtn = document.getElementById('submitLoginBtn');
const loginError = document.getElementById('loginError');

const searchInput = document.getElementById('searchInput');
const searchBtn = document.getElementById('searchBtn');
const limitInput = document.getElementById('limitInput');
const comicsGrid = document.getElementById('comicsGrid');

const adminOutput = document.getElementById('adminOutput');
const updateDbBtn = document.getElementById('updateDbBtn');
const statsBtn = document.getElementById('statsBtn');
const clearDbBtn = document.getElementById('clearDbBtn');

const imageModal = document.getElementById('imageModal');
const closeImageModal = document.getElementById('closeImageModal');
const enlargedImg = document.getElementById('enlargedImg');
const downloadBtn = document.getElementById('downloadBtn');

const searchBox = document.querySelector('.search-box');


function init() {
    updateAuthUI();
}

function updateAuthUI() {
    if (jwtToken) {
        loginBtn.classList.add('hidden');
        logoutBtn.classList.remove('hidden');

        if (currentView === 'admin') {
            mainView.classList.add('hidden');
            adminView.classList.remove('hidden');
            goToSearchBtn.classList.remove('hidden');
            goToAdminBtn.classList.add('hidden');
        } else {
            mainView.classList.remove('hidden');
            adminView.classList.add('hidden');
            goToSearchBtn.classList.add('hidden');
            goToAdminBtn.classList.remove('hidden');
        }
    } else {
        loginBtn.classList.remove('hidden');
        logoutBtn.classList.add('hidden');
        goToSearchBtn.classList.add('hidden');
        goToAdminBtn.classList.add('hidden');

        mainView.classList.remove('hidden');
        adminView.classList.add('hidden');
    }
}

function displayAdminOutput(data, title) {
    adminOutput.innerHTML = `<h3>${title}</h3><pre>${JSON.stringify(data, null, 2)}</pre>`;
}

goToAdminBtn.addEventListener('click', () => {
    currentView = 'admin';
    updateAuthUI();
});

goToSearchBtn.addEventListener('click', () => {
    currentView = 'search';
    updateAuthUI();
});


searchInput.addEventListener('input', (e) => {
    if (e.target.value.trim().length > 0) {
        searchBox.classList.add('active');
    } else {
        searchBox.classList.remove('active');
    }
});

searchBtn.addEventListener('click', async () => {
    const phrase = searchInput.value.trim();
    const limit = limitInput.value || 10;
    if (!phrase) return;

    mainView.classList.add('searched');
    comicsGrid.innerHTML = '<i>Searching...</i>';

    try {
        const res = await fetch(`${API_BASE}/api/isearch?phrase=${encodeURIComponent(phrase)}&limit=${limit}`);
        if (!res.ok) throw new Error(`Search failed with status: ${res.status}`);

        const data = await res.json();

        comicsGrid.innerHTML = '';

        if (data.comics && data.comics.length > 0) {
            data.comics.forEach(comic => {
                const item = document.createElement('div');
                item.className = 'comic-item';

                const img = document.createElement('img');
                img.src = comic.url;
                img.alt = `Comic ${comic.id}`;
                img.loading = 'lazy';

                img.addEventListener('click', () => {
                    enlargedImg.src = comic.url;
                    currentDownloadUrl = comic.url;
                    currentComicId = comic.id;
                    imageModal.classList.remove('hidden');
                });

                const label = document.createElement('div');
                label.className = 'comic-id';
                label.textContent = `#${comic.id}`;

                item.appendChild(img);
                item.appendChild(label);
                comicsGrid.appendChild(item);
            });
        } else {
            comicsGrid.innerHTML = '<p class="no-results-text">No comics found</p>';
        }
    } catch (err) {
        comicsGrid.innerHTML = `<p style="color:var(--pastel-red);">Error: ${err.message}</p>`;
    }
});

searchInput.addEventListener('keypress', (e) => {
    if (e.key === 'Enter' && searchInput.value.trim().length > 0) {
        searchBtn.click();
    }
});


loginBtn.addEventListener('click', () => {
    loginModal.classList.remove('hidden');
    loginError.classList.add('hidden');
});

closeModal.addEventListener('click', () => {
    loginModal.classList.add('hidden');
});

submitLoginBtn.addEventListener('click', async () => {
    const name = document.getElementById('usernameInput').value;
    const password = document.getElementById('passwordInput').value;

    try {
        const res = await fetch(`${API_BASE}/api/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, password })
        });

        if (!res.ok) throw new Error('Invalid login');

        const tokenString = await res.text();

        jwtToken = tokenString.trim();
        localStorage.setItem('comic_jwt', jwtToken);

        loginModal.classList.add('hidden');
        updateAuthUI();
        adminOutput.innerHTML = '';
        document.getElementById('usernameInput').value = '';
        document.getElementById('passwordInput').value = '';
    } catch (err) {
        console.error("Login Error:", err);
        loginError.classList.remove('hidden');
    }
});

logoutBtn.addEventListener('click', () => {
    jwtToken = null;
    localStorage.removeItem('comic_jwt');
    currentView = 'search';
    updateAuthUI();
});

async function adminFetch(endpoint, method = 'GET') {
    const cleanToken = jwtToken.replace(/^"|"$/g, '');

    const res = await fetch(`${API_BASE}${endpoint}`, {
        method: method,
        headers: { 'Authorization': `Token ${cleanToken}` }
    });

    if (res.status === 401 || res.status === 403) {
        throw new Error(`Server denied access (Status: ${res.status}). Check token format or backend middleware.`);
    }

    if (!res.ok) {
        throw new Error(`Request failed with status ${res.status}`);
    }

    const textData = await res.text();
    if (!textData) return {};

    try {
        return JSON.parse(textData);
    } catch (e) {
        return { message: textData };
    }
}


updateDbBtn.addEventListener('click', async () => {
    adminOutput.innerHTML = '<i>Updating database... This might take a while...</i>';
    try {
        const data = await adminFetch('/api/db/update', 'POST');
        displayAdminOutput(data, 'Database Update Result');
    } catch (err) {
        displayAdminOutput({ error: err.message }, 'Error');
    }
});

statsBtn.addEventListener('click', async () => {
    adminOutput.innerHTML = '<i>Fetching statistics...</i>';
    try {
        const data = await adminFetch('/api/db/stats', 'GET');
        displayAdminOutput(data, 'System Statistics');
    } catch (err) {
        displayAdminOutput({ error: err.message }, 'Error');
    }
});

clearDbBtn.addEventListener('click', async () => {
    if (!confirm('Are you sure you want to delete ALL comics from the database?')) return;

    adminOutput.innerHTML = '<i>Clearing database...</i>';
    try {
        await adminFetch('/api/db', 'DELETE');
        displayAdminOutput({ status: 'Success', message: 'Database cleared completely.' }, 'Clear Database');
    } catch (err) {
        displayAdminOutput({ error: err.message }, 'Error');
    }
});


closeImageModal.addEventListener('click', () => {
    imageModal.classList.add('hidden');
    enlargedImg.src = '';
});

imageModal.addEventListener('click', (e) => {
    if (e.target === imageModal) {
        imageModal.classList.add('hidden');
        enlargedImg.src = '';
    }
});

downloadBtn.addEventListener('click', async () => {
    try {
        const response = await fetch(currentDownloadUrl);
        const blob = await response.blob();

        const blobUrl = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.style.display = 'none';
        a.href = blobUrl;
        a.download = `comic_${currentComicId}.png`;

        document.body.appendChild(a);
        a.click();

        window.URL.revokeObjectURL(blobUrl);
        a.remove();
    } catch (err) {
        console.warn("Fetch blocked by CORS, fallback to direct link open.");
        const a = document.createElement('a');
        a.href = currentDownloadUrl;
        a.target = '_blank';
        a.download = `comic_${currentComicId}.png`;
        a.click();
    }
});

init();