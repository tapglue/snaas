module Loader exposing (nextStep, view)

import Color exposing (Color, toRgb)
import Html exposing (Html, div)
import Svg exposing (g, rect, svg)
import Svg.Attributes exposing (class, fill, height, style, transform, viewBox, width, x, y)
import Time exposing (Time)


-- MODEL


type alias Square =
    { color : Color
    , opacity : Float
    , scale : Float
    , height : Int
    , width : Int
    , tx : Float
    , ty : Float
    , x : Int
    , y : Int
    }



-- VIEW


view : Int -> Color -> Int -> Html msg
view size color step =
    let
        s =
            toString size
    in
        div [ class "loader" ]
            [ svg
                [ height (s ++ "px")
                , width (s ++ "px")
                , viewBox ("0 0 " ++ s ++ " " ++ s)
                ]
                [ g [] (List.map viewSquare (squares size color step))
                ]
            ]


viewSquare : Square -> Svg.Svg msg
viewSquare square =
    let
        color =
            toRgb square.color
    in
        rect
            [ fill ("rgb(" ++ toString color.red ++ ", " ++ toString color.green ++ ", " ++ toString color.blue ++ ")")
            , height (toString square.height)
            , width (toString square.width)
            , transform ("translate(" ++ (toString square.tx) ++ " " ++ (toString square.ty) ++ ") scale(" ++ (toString square.scale) ++ ")")
            , style ("opacity: " ++ (toString square.opacity) ++ ";")
            , x (toString square.x)
            , y (toString square.y)
            ]
            []



-- ANIMATION


animationLength : Int
animationLength =
    1225



-- HELPER


nextStep : Time -> Time -> Int
nextStep startTime time =
    (floor time - floor startTime) % animationLength


squareX : Int -> Int -> Int
squareX size index =
    if index % 2 == 0 then
        0
    else
        size // 2


squareY : Int -> Int -> Int
squareY size index =
    if index == 0 then
        0
    else if index == 1 then
        0
    else
        size // 2


squareTx : Int -> Float -> Int -> Float
squareTx size scale index =
    let
        modifier =
            if index % 2 == 0 then
                toFloat size * 0.25
            else
                (toFloat size * 0.25) + (toFloat size * 0.5)
    in
        (1 - scale) * modifier


squareTy : Int -> Float -> Int -> Float
squareTy size scale index =
    let
        modifier =
            if index < 2 then
                toFloat size * 0.25
            else
                (toFloat size * 0.25) + (toFloat size * 0.5)
    in
        (1 - scale) * modifier


computeSquare : Int -> Color -> Int -> Int -> Square
computeSquare size color step index =
    let
        a =
            min (toFloat step / toFloat 48) (toFloat 48)

        magic =
            1 + 0.15 * toFloat index

        scale =
            min (abs (magic - a / 10)) 1

        opacity =
            0.6 * scale + 0.4

        tx =
            squareTx size scale index

        ty =
            squareTy size scale index

        height =
            size // 2

        width =
            size // 2
    in
        Square color opacity scale height width tx ty (squareX size index) (squareY size index)


squares : Int -> Color -> Int -> List Square
squares size color step =
    List.map (computeSquare size color step) (List.range 0 3)
