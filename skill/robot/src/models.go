package examples

import (
	"image"
	"time"
)

/* View */
type View struct {
	id        int64
	name      string
	image     *image.RGBA
	direction float64
	angle     float64
	timestamp time.Time
}

func NewView(name string, image *image.RGBA, direction float64, angle float64, timestamp time.Time) View {
	return View{
		id:        time.Now().Unix(),
		name:      name,
		image:     image,
		direction: direction,
		angle:     angle,
		timestamp: time.Now(),
	}

}

//func (view View) UpdateImage(){

//}

// func (view View) LookAt(){

// }
