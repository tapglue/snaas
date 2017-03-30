module Tests exposing (all)

import Expect
import Test exposing (..)
import LocalStorage


all : Test
all =
    describe "LocalStorage"
        [ test "Retrieving non-existent key" <|
            \() ->
                case LocalStorage.get "nonexistentKey" of
                    Err err ->
                        case err of
                            LocalStorage.NotFound ->
                                Expect.pass

                            _ ->
                                Expect.fail "should result in NotFound"

                    Ok _ ->
                        Expect.fail "shouldn't be Ok"
        , test "storing and retrieving a key" <|
            \() ->
                case LocalStorage.set "foo" "bar" of
                    Err err ->
                        Expect.fail ("should not return " ++ (toString err))

                    Ok _ ->
                        case LocalStorage.get "foo" of
                            Err err ->
                                Expect.fail (toString err)

                            Ok val ->
                                Expect.equal val "bar"
        ]
