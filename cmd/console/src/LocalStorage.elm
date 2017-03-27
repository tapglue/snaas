module LocalStorage exposing
    ( Error(..)
    , clear
    , get
    , keys
    , remove
    , set
    )

import Native.LocalStorage


type Error
    = Disabled
    | InvalidValue
    | NotFound
    | QuotaExceeded


clear : Result Error ()
clear =
    Native.LocalStorage.clear

get : String -> Result Error String
get =
    Native.LocalStorage.get

keys : Result Error (List String)
keys =
    Native.LocalStorage.keys

remove : String -> Result Error ()
remove =
    Native.LocalStorage.clear

set : String -> String -> Result Error ()
set =
    Native.LocalStorage.set