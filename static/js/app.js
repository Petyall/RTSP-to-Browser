class StreamViewer {
  constructor() {
    this.streams = [];
    this.players = new Map();
    this.init();
  }

  async fetchCameras() {
    const response = await fetch('/api/cameras');
    this.streams = await response.json();
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

    player.tech(true).setAttribute('crossorigin', 'anonymous');
    player.tech(true).setAttribute('preload', 'none');

    this.players.set(stream.id, player);
  }

  async init() {
    await this.fetchCameras();
    const grid = document.getElementById('videoGrid');
    
    this.streams.forEach(stream => {
      const card = this.createVideoCard(stream);
      grid.appendChild(card);
      this.initializePlayer(stream);
    });
  }
}

document.addEventListener('DOMContentLoaded', () => {
  new StreamViewer();
});
