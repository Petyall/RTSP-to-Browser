package database

type Camera struct {
	ID      int    `json:"id"`
	RtspURL string `json:"rtsp_url"`
}

type CameraAPI struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}
