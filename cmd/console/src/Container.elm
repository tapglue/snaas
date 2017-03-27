module Container exposing (..)

import Html exposing (Html, div)
import Html.Attributes exposing (class)


view : (List (Html msg) -> Html msg) -> List (Html msg) -> Html msg
view elem content =
    elem
        [ div [ class "container" ]
            content
        ]
