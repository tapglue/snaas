module User.Model exposing (User, decode, decodeList, encode, initUserSearchForm, initUserUpdateForm)

import Json.Decode as Decode
import Json.Encode as Encode
import RemoteData exposing (RemoteData(Success), WebData)
import Formo exposing (Form, initForm, updateElementValue, validatorExist, validatorMinLength)


-- MODEL


type alias User =
    { email : String
    , enabled : Bool
    , id : String
    , username : String
    }



-- DECODERS


decode : Decode.Decoder User
decode =
    Decode.map4 User
        (Decode.field "email" Decode.string)
        (Decode.field "enabled" Decode.bool)
        (Decode.field "id_string" Decode.string)
        (Decode.field "user_name" Decode.string)


decodeList : Decode.Decoder (List User)
decodeList =
    Decode.at [ "users" ] (Decode.list decode)


encode : String -> Encode.Value
encode username =
    Encode.object
        [ ( "user_name", Encode.string username )
        ]



-- FORM


initUserSearchForm : Form
initUserSearchForm =
    initForm
        [ ( "query"
          , [ validatorExist
            , validatorMinLength 3
            ]
          )
        ]


initUserUpdateForm : WebData User -> Form
initUserUpdateForm user =
    let
        form =
            initForm
                [ ( "username"
                  , [ validatorExist
                    , validatorMinLength 3
                    ]
                  )
                ]
    in
        case user of
            Success user ->
                updateElementValue form "username" user.username

            _ ->
                form
