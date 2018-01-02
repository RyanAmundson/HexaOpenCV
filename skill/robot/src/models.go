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

func NewView(name string, direction float64, angle float64) View {
	return View{
		id:        time.Now().Unix(),
		name:      name,
		image:     TakePicAndSend(),
		direction: direction,
		angle:     angle,
		timestamp: time.Now(),
	}

}

func NewViewWithImage(name string, image *image.RGBA, direction float64, angle float64) View {
	return View{
		id:        time.Now().Unix(),
		name:      name,
		image:     image,
		direction: direction,
		angle:     angle,
		timestamp: time.Now(),
	}

}

// func (view View) UpdateView(){

// }

// func (view View) LookAt(){

// }

// func (view View) ContainsFace(){

// }
