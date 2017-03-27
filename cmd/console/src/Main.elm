module Main exposing (main)

import AnimationFrame
import Navigation
import Action exposing (Msg(..))
import Ask exposing (confirmRuleActivate, confirmRuleDeactivate, confirmRuleDelete)
import Model exposing (Flags, Model, init)
import Update exposing (update)
import View exposing (view)


main : Program Flags Model Action.Msg
main =
    Navigation.programWithFlags LocationChange
        { init = init
        , subscriptions = subscriptions
        , update = update
        , view = view
        }


-- SUBSCRIPTION

subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.batch
        [ AnimationFrame.times Tick
        , confirmRuleActivate RuleActivateConfirm
        , confirmRuleDeactivate RuleDeactivateConfirm
        , confirmRuleDelete RuleDeleteConfirm
        ]
