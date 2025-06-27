package database

type Camera struct {
	ID      int    `gorm:"primaryKey" json:"id"`
	RtspURL string `gorm:"column:rtsp_url;type:varchar(255);not null" json:"rtsp_url"`
	Name    string `gorm:"column:name;type:varchar(100);not null" json:"name"`
}

type CameraAPI struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}
