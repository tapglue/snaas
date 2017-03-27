module Member.Model exposing (Member, decode, encode, encodeAuth)

import Json.Decode as Decode
import Json.Encode as Encode


type alias Auth =
    { accessToken : String
    }

type alias Member =
    { auth : Auth
    , name : String
    , picture : String
    }

decode : Decode.Decoder Member
decode =
    Decode.map3 Member
        (Decode.field "auth" decodeAuth)
        (Decode.field "name" Decode.string)
        (Decode.field "picture" Decode.string)

decodeAuth : Decode.Decoder Auth
decodeAuth =
    Decode.map Auth
        (Decode.field "access_token" Decode.string)

encode : String -> Encode.Value
encode code =
    Encode.object
        [ ( "code", Encode.string code )
        ]

encodeAuth : String -> Encode.Value
encodeAuth token =
    Encode.object
        [ ( "access_token", Encode.string token )
        ]