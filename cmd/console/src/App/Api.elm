module App.Api exposing (createApp, getApp, getApps)

import Http
import RemoteData exposing (WebData, sendRequest)
import App.Model exposing (App, decode, decodeList, encode)


createApp : String -> String -> Cmd (WebData App)
createApp name description =
    Http.post "/api/apps" (encode name description) decode
        |> sendRequest


getApp : String -> Cmd (WebData App)
getApp id =
    Http.get ("/api/apps/" ++ id) decode
        |> sendRequest


getApps : Cmd (WebData (List App))
getApps =
    Http.get "/api/apps" decodeList
        |> sendRequest
