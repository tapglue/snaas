module Model exposing (Flags, Model, init, initRoute, isLoggedIn)

import Navigation
import RemoteData exposing (RemoteData(..), WebData)
import Time exposing (Time)
import Action exposing (..)
import App.Api exposing (getApp, getApps)
import App.Model exposing (App, initAppForm)
import Formo exposing (Form)
import LocalStorage
import Member.Api exposing (fetchMember, loginMember)
import Member.Model exposing (Member)
import Route exposing (Route, parse)
import Rule.Api exposing (getRule, listRules)
import Rule.Model exposing (Rule)


type alias Flags =
    { loginUrl : String
    , zone : String
    }


type alias Model =
    { app : WebData App
    , apps : WebData (List App)
    , appForm : Form
    , appId : String
    , focus : String
    , loginUrl : String
    , member : WebData Member
    , newApp : WebData App
    , route : Maybe Route
    , rule : WebData Rule
    , rules : WebData (List Rule)
    , startTime : Time
    , time : Time
    , zone : String
    }


init : Flags -> Navigation.Location -> ( Model, Cmd Msg )
init { loginUrl, zone } location =
    let
        route =
            parse location

        model =
            initModel loginUrl zone route
    in
        case route of
            Just (Route.OAuthCallback code _) ->
                case code of
                    Nothing ->
                        ( model, Cmd.map LocationChange (Route.navigate Route.Login) )

                    Just code ->
                        ( model, Cmd.map MemberLogin (loginMember code) )

            Just (Route.Login) ->
                ( model, Cmd.none )

            _ ->
                case LocalStorage.get "token" of
                    Err _ ->
                        ( model, Cmd.map LocationChange (Route.navigate Route.Login) )

                    Ok token ->
                        ( model, Cmd.map MemberFetch (fetchMember token) )


initModel : String -> String -> Maybe Route -> Model
initModel loginUrl zone route =
    Model NotAsked NotAsked initAppForm "" "" loginUrl Loading NotAsked route NotAsked NotAsked 0 0 zone


initRoute : Model -> ( Model, Cmd Msg )
initRoute model =
    case model.route of
        Just (Route.App id) ->
            ( { model | app = Loading, appId = id }, Cmd.map FetchApp (getApp id) )

        Just (Route.Apps) ->
            ( { model | apps = Loading }, Cmd.map FetchApps getApps )

        Just (Route.Rules appId) ->
            case model.app of
                Success _ ->
                    ( { model | appId = appId, rule = NotAsked, rules = Loading }
                    , Cmd.batch
                        [ Cmd.map FetchRules (listRules appId)
                        ]
                    )

                _ ->
                    ( { model | app = Loading, appId = appId, rules = Loading }
                    , Cmd.batch
                        [ Cmd.map FetchApp (getApp appId)
                        , Cmd.map FetchRules (listRules appId)
                        ]
                    )

        Just (Route.Rule appId ruleId) ->
            case model.app of
                Success _ ->
                    ( { model | appId = appId, rule = Loading }
                    , Cmd.batch
                        [ Cmd.map FetchRule (getRule appId ruleId)
                        ]
                    )

                _ ->
                    ( { model | app = Loading, appId = appId, rule = Loading }
                    , Cmd.batch
                        [ Cmd.map FetchApp (getApp appId)
                        , Cmd.map FetchRule (getRule appId ruleId)
                        ]
                    )

        _ ->
            ( model, Cmd.none )


isLoggedIn : Model -> Bool
isLoggedIn model =
    case model.member of
        Success _ ->
            True

        _ ->
            False
