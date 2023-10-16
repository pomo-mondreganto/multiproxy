package ports

import (
	"fmt"
	"strconv"
	"strings"
)

type Definition struct {
	SourceStart uint16
	SourceEnd   uint16
	DestStart   uint16
	DestEnd     uint16
}

func Parse(def string) (*Definition, error) {
	tokens := strings.Split(def, ":")
	if len(tokens) != 2 {
		return nil, fmt.Errorf("invalid ports definition: %s", def)
	}
	startTokens := strings.Split(tokens[0], "-")
	if len(startTokens) != 2 {
		return nil, fmt.Errorf("invalid ports definition: %s", def)
	}
	endTokens := strings.Split(tokens[1], "-")
	if len(endTokens) != 2 {
		return nil, fmt.Errorf("invalid ports definition: %s", def)
	}

	sourceStart, err := strconv.ParseUint(startTokens[0], 10, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid ports definition: %w", err)
	}

	sourceEnd, err := strconv.ParseUint(startTokens[1], 10, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid ports definition: %w", err)
	}

	destStart, err := strconv.ParseUint(endTokens[0], 10, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid ports definition: %w", err)
	}

	destEnd, err := strconv.ParseUint(endTokens[1], 10, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid ports definition: %w", err)
	}

	if sourceStart > sourceEnd {
		return nil, fmt.Errorf("invalid ports definition: source start > source end")
	}

	if destStart > destEnd {
		return nil, fmt.Errorf("invalid ports definition: dest start > dest end")
	}

	if sourceEnd-sourceStart != destEnd-destStart {
		return nil, fmt.Errorf("invalid ports definition: source range != dest range")
	}

	return &Definition{
		SourceStart: uint16(sourceStart),
		SourceEnd:   uint16(sourceEnd),
		DestStart:   uint16(destStart),
		DestEnd:     uint16(destEnd),
	}, nil
}
