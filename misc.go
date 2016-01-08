package main

import(
    "image"
    "image/jpeg"
)

func init(){
    image.RegisterFormat("jpg", "\xff\xd8", jpeg.Decode, jpeg.DecodeConfig)
}