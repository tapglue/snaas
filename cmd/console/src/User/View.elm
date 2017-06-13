module User.View exposing (viewUser, viewUserItem, viewUserTable)

import Color exposing (rgb)
import Json.Decode as Decode
import Html exposing (Html, h3, li, span, strong, table, tbody, td, text, th, thead, tr, ul)
import Html.Attributes exposing (class)
import Html.Events exposing (onClick)
import User.Model exposing (User)


viewEnabled : Bool -> Html msg
viewEnabled enabled =
    if enabled then
        span [ class "nc-icon-outline ui-1_check-circle-08" ] []
    else
        span [ class "nc-icon-outline ui-1_circle-remove" ] []


viewUser : User -> Html msg
viewUser user =
    h3 []
        [ text "This is the user"
        , strong [] [ text user.username ]
        , text "with the ID"
        , strong [] [ text user.id ]
        , text "and email"
        , strong [] [ text user.email ]
        ]


viewUserItem : msg -> User -> Html msg
viewUserItem msg user =
    tr [ onClick msg ]
        [ td [ class "icon" ] [ viewEnabled user.enabled ]
        , td [] [ text user.username ]
        , td [] [ text user.id ]
        ]


viewUserTable : (User -> Html msg) -> List User -> Html msg
viewUserTable item users =
    table [ class "navigation" ]
        [ thead []
            [ tr []
                [ th [ class "icon" ] [ text "enabled" ]
                , th [] [ text "username" ]
                , th [] [ text "id" ]
                ]
            ]
        , tbody [] (List.map item users)
        ]
