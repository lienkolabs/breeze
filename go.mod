module github.com/lienkolabs/breeze

go 1.19

replace github.com/lienkolabs/papirus => ../papirus

replace github.com/lienkolabs/echo => ../echo

replace github.com/lienkolabs/swell => ../swell

require (
	github.com/lienkolabs/papirus v0.0.0-00010101000000-000000000000
	golang.org/x/term v0.9.0
)

require golang.org/x/sys v0.9.0 // indirect
