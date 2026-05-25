package rest

import "where-is-my-comic-service/search-services/api/core"

func MapToComicsDTO(comics core.Comics) ComicsDTO {
	return ComicsDTO{
		ID:  comics.ID,
		URL: comics.URL,
	}
}

func MapToSliceComicsDTO(comics []core.Comics) []ComicsDTO {
	comicsDTO := make([]ComicsDTO, len(comics))
	for i, c := range comics {
		comicsDTO[i] = MapToComicsDTO(c)
	}

	return comicsDTO
}
