module App.View exposing (viewAppItem, viewAppsTable)

import Html exposing (Html, a, div, h2, input, nav, section, small, span, table, tbody, td, th, thead, tr, text)
import Html.Attributes exposing (class, id, placeholder, title, type_, value)
import Html.Events exposing (onClick, onInput)
import App.Model exposing (App)


viewAppItem : msg -> App -> Html msg
viewAppItem msg app =
    let
        enabled =
            if app.enabled then
                span [ class "nc-icon-glyph ui-1_check-circle-07" ] []
            else
                span [ class "nc-icon-glyph ui-1_circle-remove" ] []
    in
        tr [ onClick msg ]
            [ td [ class "icon" ] [ enabled ]
            , td [] [ text app.name ]
            , td [] [ text app.description ]
            , td [] [ text app.token ]
            ]


viewAppsTable : (App -> Html msg) -> List App -> Html msg
viewAppsTable item apps =
    table [ class "navigation" ]
        [ thead []
            [ tr []
                [ th [ class "icon" ] [ text "status" ]
                , th [] [ text "name" ]
                , th [] [ text "description" ]
                , th [] [ text "token" ]
                ]
            ]
        , tbody [] (List.map item apps)
        ]
