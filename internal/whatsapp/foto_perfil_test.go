package whatsapp

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestPrepararFotoPerfilRecortaEReduz(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1200, 800))
	for y := 0; y < 800; y++ {
		for x := 0; x < 1200; x++ {
			img.Set(x, y, color.NRGBA{R: 200, G: 10, B: 10, A: 255})
		}
	}
	var png1 bytes.Buffer
	_ = png.Encode(&png1, img)
	saida, err := PrepararFotoPerfil(png1.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	foto, err := jpeg.Decode(bytes.NewReader(saida))
	if err != nil {
		t.Fatalf("saida nao e JPEG: %v", err)
	}
	if b := foto.Bounds(); b.Dx() != 640 || b.Dy() != 640 {
		t.Fatalf("tamanho %v, esperado 640x640", b)
	}
	if r, _, _, _ := foto.At(320, 320).RGBA(); r>>8 < 180 {
		t.Fatalf("cor perdida: r=%d", r>>8)
	}
}

func TestPrepararFotoPerfilPequenaMantemTamanho(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 300))
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, nil)
	saida, err := PrepararFotoPerfil(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	foto, _ := jpeg.Decode(bytes.NewReader(saida))
	if b := foto.Bounds(); b.Dx() != 100 || b.Dy() != 100 {
		t.Fatalf("tamanho %v, esperado 100x100", b)
	}
}

func TestPrepararFotoPerfilRecusaLixo(t *testing.T) {
	if _, err := PrepararFotoPerfil([]byte("nao e imagem")); !errors.Is(err, ErrMidiaInvalida) {
		t.Fatalf("esperado ErrMidiaInvalida, veio %v", err)
	}
}
