port module Ask exposing
    ( askRuleActivate
    , askRuleDeactivate
    , askRuleDelete
    , confirmRuleActivate
    , confirmRuleDeactivate
    , confirmRuleDelete
    )

port askRuleActivate : String -> Cmd msg

port askRuleDeactivate : String -> Cmd msg

port askRuleDelete : String -> Cmd msg

port confirmRuleActivate : ( String -> msg ) -> Sub msg

port confirmRuleDeactivate : ( String -> msg ) -> Sub msg

port confirmRuleDelete : ( String -> msg ) -> Sub msg
