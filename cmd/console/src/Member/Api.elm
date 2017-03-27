module Member.Api exposing (fetchMember, loginMember)

import Http
import RemoteData exposing (WebData, sendRequest)
import Member.Model exposing (Member, decode, encode, encodeAuth)


fetchMember : String -> Cmd (WebData Member)
fetchMember token =
    Http.post "/api/me" (Http.jsonBody (encodeAuth token)) decode
        |> sendRequest

loginMember : String -> Cmd (WebData Member)
loginMember code =
    Http.post "/api/me/login" (Http.jsonBody (encode code)) decode
        |> sendRequest
