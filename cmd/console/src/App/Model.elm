module App.Model exposing (App, decode, decodeList, encode, initAppForm)

import Http
import Json.Decode as Decode
import Json.Encode as Encode
import Formo exposing (Form, initForm, validatorExist, validatorMaxLength, validatorMinLength)


-- MODEL


type alias App =
    { backend_token : String
    , counts : Counts
    , description : String
    , enabled : Bool
    , id : String
    , name : String
    , token : String
    }


type alias Counts =
    { comments : Int
    , connections: Int
    , devices : Int
    , posts : Int
    , rules : Int
    , users : Int
    }



-- DECODERS


decode : Decode.Decoder App
decode =
    Decode.map7 App
        (Decode.field "backend_token" Decode.string)
        (Decode.field "counts" decodeCounts)
        (Decode.field "description" Decode.string)
        (Decode.field "enabled" Decode.bool)
        (Decode.field "id" Decode.string)
        (Decode.field "name" Decode.string)
        (Decode.field "token" Decode.string)


decodeCounts : Decode.Decoder Counts
decodeCounts =
    Decode.map6 Counts
        (Decode.field "comments" Decode.int)
        (Decode.field "connections" Decode.int)
        (Decode.field "devices" Decode.int)
        (Decode.field "posts" Decode.int)
        (Decode.field "rules" Decode.int)
        (Decode.field "users" Decode.int)


decodeList : Decode.Decoder (List App)
decodeList =
    Decode.at [ "apps" ] (Decode.list decode)


encode : String -> String -> Http.Body
encode name description =
    Encode.object
        [ ( "name", Encode.string name )
        , ( "description", Encode.string description )
        ]
        |> Http.jsonBody



-- FORM


initAppForm : Form
initAppForm =
    initForm
        [ ( "description"
          , [ validatorExist
            , validatorMaxLength 42
            , validatorMinLength 12
            ]
          )
        , ( "name"
          , [ validatorExist
            , validatorMaxLength 16
            , validatorMinLength 3
            ]
          )
        ]
