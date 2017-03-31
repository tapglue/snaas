module Update exposing (update)

import RemoteData exposing (RemoteData(Failure, Loading, NotAsked, Success), WebData)
import Task
import Action exposing (Msg(..))
import Confirm
import Formo exposing (blurElement, elementValue, focusElement, updateElementValue, validateForm)
import LocalStorage
import Model exposing (Flags, Model, init, initRoute)
import App.Api exposing (createApp)
import App.Model exposing (initAppForm)
import Route
import Rule.Api exposing (activateRule, deactivateRule, deleteRule)


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        AppFormBlur field ->
            ( { model | appForm = blurElement model.appForm field }, Cmd.none )

        AppFormClear ->
            ( { model | appForm = initAppForm }, Cmd.none )

        AppFormFocus field ->
            ( { model | appForm = focusElement model.appForm field }, Cmd.none )

        AppFormSubmit ->
            let
                ( form, isValid ) =
                    validateForm model.appForm
            in
                case isValid of
                    True ->
                        ( { model | newApp = Loading }, Cmd.map NewApp (createApp (elementValue model.appForm "name") (elementValue model.appForm "description")) )

                    False ->
                        ( { model | appForm = form }, Cmd.none )

        AppFormUpdate field value ->
            ( { model | appForm = updateElementValue model.appForm field value }, Cmd.none )

        FetchApp response ->
            ( { model | app = response }, Cmd.none )

        FetchApps response ->
            ( { model | app = NotAsked, apps = response }, Cmd.none )

        FetchRule response ->
            ( { model | rule = response }, Cmd.none )

        FetchRules response ->
            ( { model | rules = response }, Cmd.none )

        LocationChange location ->
            initRoute { model | route = (Route.parse location) }

        MemberFetch member ->
            case member of
                NotAsked ->
                    ( { model | member = member }, Cmd.none )

                Loading ->
                    ( { model | member = member }, Cmd.none )

                Failure _ ->
                    ( model, Cmd.map LocationChange (Route.navigate Route.Login) )

                Success _ ->
                    initRoute { model | member = member }

        MemberLogin data ->
            case data of
                NotAsked ->
                    ( { model | member = data }, Cmd.none )

                Loading ->
                    ( { model | member = data }, Cmd.none )

                Failure _ ->
                    ( model, Cmd.map LocationChange (Route.navigate Route.Login) )

                Success member ->
                    ( { model | member = data }
                    , Cmd.batch
                        [ Task.perform TokenPersist (saveToken member.auth.accessToken)
                        , Cmd.map LocationChange (Route.navigate Route.Dashboard)
                        ]
                    )

        Navigate route ->
            ( model, Cmd.map LocationChange (Route.navigate route) )

        NewApp response ->
            ( { model | appForm = initAppForm, apps = (appendWebData model.apps response), newApp = NotAsked }, Cmd.none )

        RuleActivate (Err _) ->
            ( model, Cmd.none )

        RuleActivate (Ok id) ->
            ( model, Cmd.map LocationChange (Route.navigate (Route.Rule model.appId id)) )

        RuleActivateAsk id ->
            ( model, guardConfirm "You want to activate this ruel?" (activateRule RuleActivate model.appId id) )

        RuleDeactivate (Err _) ->
            ( model, Cmd.none )

        RuleDeactivate (Ok id) ->
            ( model, Cmd.map LocationChange (Route.navigate (Route.Rule model.appId id)) )

        RuleDeactivateAsk id ->
            ( model, guardConfirm "You want to deactivate this rule?" (deactivateRule RuleDeactivate model.appId id) )

        RuleDeleteAsk id ->
            ( model, guardConfirm "You want to delete this rule?" (deleteRule RuleDelete model.appId id) )

        RuleDelete _ ->
            ( model, Cmd.map LocationChange (Route.navigate (Route.Rules model.appId)) )

        TokenPersist _ ->
            ( model, Cmd.none )

        Tick time ->
            let
                startTime =
                    if model.startTime == 0 then
                        time
                    else
                        model.startTime
            in
                ( { model | startTime = startTime, time = time }, Cmd.none )



-- HELPER


appendWebData : WebData (List a) -> WebData a -> WebData (List a)
appendWebData list single =
    case (RemoteData.toMaybe single) of
        Nothing ->
            list

        Just a ->
            case (RemoteData.toMaybe list) of
                Nothing ->
                    RemoteData.succeed [ a ]

                Just list ->
                    RemoteData.succeed (list ++ [ a ])


guardConfirm : String -> Cmd Msg -> Cmd Msg
guardConfirm question cmd =
    case Confirm.dialog question of
        Err _ ->
            Cmd.none

        Ok confirm ->
            if confirm then
                cmd
            else
                Cmd.none


saveToken : String -> Task.Task Never String
saveToken token =
    case (LocalStorage.set "token" token) of
        _ ->
            Task.succeed token
