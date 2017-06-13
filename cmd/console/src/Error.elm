module Error exposing (decodeList)

import Json.Decode as Decode


type alias Error =
    { code : Int
    , message : String
    }


decode : Decode.Decoder Error
decode =
    Decode.map2 Error
        (Decode.field "code" Decode.int)
        (Decode.field "message" Decode.string)


decodeList : Decode.Decoder (List Error)
decodeList =
    Decode.at [ "errors" ] (Decode.list decode)
