port module Main exposing (..)

import Tests
import Test.Runner.Node exposing (TestProgram, run)
import Json.Encode exposing (Value)


main : TestProgram
main =
    run emit Tests.all


port emit : ( String, Value ) -> Cmd msg
