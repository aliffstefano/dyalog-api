package whatsapp

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/gif" // decodificadores aceitos na foto de perfil
	"image/jpeg"
	_ "image/png"
)

// ladoFotoPerfil e o tamanho que o app oficial envia (640x640).
const ladoFotoPerfil = 640

// PrepararFotoPerfil converte JPEG, PNG ou GIF numa foto de perfil aceita pelo
// WhatsApp: recorta o centro em quadrado, reduz para no maximo 640x640 e
// regrava em JPEG. O servidor recusa imagens que nao sejam JPEG.
func PrepararFotoPerfil(dados []byte) ([]byte, error) {
	original, _, err := image.Decode(bytes.NewReader(dados))
	if err != nil {
		return nil, fmt.Errorf("%w: imagem nao reconhecida (use JPG, PNG ou GIF)", ErrMidiaInvalida)
	}
	limites := original.Bounds()
	lado := min(limites.Dx(), limites.Dy())
	if lado < 32 {
		return nil, fmt.Errorf("%w: imagem pequena demais", ErrMidiaInvalida)
	}
	x0 := limites.Min.X + (limites.Dx()-lado)/2
	y0 := limites.Min.Y + (limites.Dy()-lado)/2
	destino := min(lado, ladoFotoPerfil)
	saida := image.NewRGBA(image.Rect(0, 0, destino, destino))
	// Media da area de origem de cada pixel: reduz sem serrilhado e sem
	// dependencia externa.
	escala := float64(lado) / float64(destino)
	for y := 0; y < destino; y++ {
		sy0 := y0 + int(float64(y)*escala)
		sy1 := max(sy0+1, y0+int(float64(y+1)*escala))
		for x := 0; x < destino; x++ {
			sx0 := x0 + int(float64(x)*escala)
			sx1 := max(sx0+1, x0+int(float64(x+1)*escala))
			var r, g, b, n uint64
			for sy := sy0; sy < sy1; sy++ {
				for sx := sx0; sx < sx1; sx++ {
					cr, cg, cb, ca := original.At(sx, sy).RGBA()
					// Transparencia vira fundo branco.
					branco := uint64(0xffff - ca)
					r += uint64(cr) + branco
					g += uint64(cg) + branco
					b += uint64(cb) + branco
					n++
				}
			}
			saida.SetRGBA(x, y, color.RGBA{R: uint8(r / n >> 8), G: uint8(g / n >> 8), B: uint8(b / n >> 8), A: 0xff})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, saida, &jpeg.Options{Quality: 88}); err != nil {
		return nil, fmt.Errorf("erro ao gerar JPEG: %w", err)
	}
	return buf.Bytes(), nil
}
