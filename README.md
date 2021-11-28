# stegno

- steganography tool that hides data into a png file at the moment we only
  support png images but in the future we will add more image formats. also we
don't support data encryption for now.



## Quick Start

build the program first.
```console
$ go build
```

```console
$ ./stegno --encode --png <image path> --to <out file>  --data <message> or --file <file path>
$ ./stegno --decode --png <image path> --to <file path> or --dump true
```


## TODO list

- [ ] Add support for data encryption and decryption before encoding or
  decoding data into the image.

- [ ] code cleaning.
