class StreamViewer {
  constructor() {
    this.streams = [];
    this.players = new Map();
    
    this.init();
  }

  async fetchCameras() {
    try {
      const response = await fetch('/api/cameras');
      if (!response.ok) {
        throw new Error(`Ошибка API: ${response.status}`);
      }
      this.streams = await response.json();
    } catch (error) {
      console.error('Ошибка при загрузке камер:', error);
      const grid = document.getElementById('videoGrid');
      grid.innerHTML = '<div class="text-center text-danger">Не удалось загрузить список камер. Попробуйте позже.</div>';
    }
  }

  createVideoElement(stream) {
    const videoElement = document.createElement('video');
    videoElement.className = 'video-js vjs-default-skin';
    videoElement.setAttribute('id', `video-${stream.id}`);
    return videoElement;
  }

  createVideoCard(stream) {
    const col = document.createElement('div');
    col.className = 'col-xl-3 col-lg-4 col-md-6';
    
    const card = document.createElement('div');
    card.className = 'video-card';
    card.innerHTML = `
      <h5 class="card-title">${stream.name}</h5>
      <div class="video-container"></div>
    `;

    const videoElement = this.createVideoElement(stream);
    card.querySelector('.video-container').appendChild(videoElement);
    col.appendChild(card);
    
    return col;
  }

  initializePlayer(stream) {
    const nocacheUrl = `${stream.url}?nocache=${new Date().getTime()}`;

    const player = videojs(`video-${stream.id}`, {
      autoplay: true,
      controls: true,
      muted: true,
      fluid: true,
      html5: {
        vhs: {
          overrideNative: true,
          cacheEncryptionKeys: false,
          limitRenditionByPlayerDimensions: false,
        }
      }
    });

    player.src({
      src: nocacheUrl,
      type: 'application/x-mpegURL'
    });

    player.on('error', () => {
      const errorDisplay = player.el().querySelector('.vjs-error-display .vjs-modal-dialog-content');
      if (errorDisplay) {
        errorDisplay.innerHTML = '<div>Камера недоступна, проверьте ваше подключение или попробуйте позже...</div>';
      }
    });

    player.tech(true).setAttribute('crossorigin', 'anonymous');
    player.tech(true).setAttribute('preload', 'none');

    this.players.set(stream.id, player);
  }

  destroy() {
    this.players.forEach(player => {
      player.dispose();
    });
    this.players.clear();
  }

  async init() {
    await this.fetchCameras();
    if (this.streams.length === 0) return;

    this.streams.sort((a, b) => a.name.localeCompare(b.name, 'ru'));

    const grid = document.getElementById('videoGrid');
    const cards = this.streams.map(stream => this.createVideoCard(stream));
    cards.forEach(card => grid.appendChild(card));

    // Вызываем setCardTitleHeights после добавления карточек
    if (typeof setCardTitleHeights === 'function') {
      setCardTitleHeights();
    }

    await Promise.all(this.streams.map(stream => this.initializePlayer(stream)));
  }
}

document.addEventListener('DOMContentLoaded', () => {
  const viewer = new StreamViewer();
  window.addEventListener('beforeunload', () => viewer.destroy());
});