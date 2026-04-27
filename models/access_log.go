package models

import "time"

type AccessLog struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	DoorID     string    `json:"door_id"`
	AccessTime time.Time `json:"access_time"`
	Granted    bool      `json:"granted"`
}
