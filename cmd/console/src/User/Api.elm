module User.Api exposing (getUser, searchUser, updateUser)

import Http
import RemoteData exposing (WebData, sendRequest)
import User.Model exposing (User, decode, decodeList, encode)


getUser : String -> String -> Cmd (WebData User)
getUser appId userId =
    Http.get ("/api/apps/" ++ appId ++ "/users/" ++ userId) decode
        |> sendRequest


searchUser : String -> String -> Cmd (WebData (List User))
searchUser appId query =
    Http.get ("/api/apps/" ++ appId ++ "/users/search?q=" ++ query) decodeList
        |> sendRequest


updateUser : String -> String -> String -> Cmd (WebData User)
updateUser appId userId username =
    Http.request
        { body = (encode username |> Http.jsonBody)
        , expect = Http.expectJson decode
        , headers = []
        , method = "PUT"
        , timeout = Nothing
        , url = (userUrl appId userId)
        , withCredentials = False
        }
        |> sendRequest


userUrl : String -> String -> String
userUrl appId userId =
    "/api/apps/" ++ appId ++ "/users/" ++ userId
