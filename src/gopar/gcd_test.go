package main

import (
    "testing"
    )
    
func TestSimpleGcd (t *testing.T){
    if Gcd(48, 18) != 6 {
        t.Fail()
    
    }
    
    if Gcd(13, 8) != 1{
        t.Fail()
     
    }
    
    if Gcd(2, 2) != 2 {
        t.Fail()
    }
    
    if Gcd(2, 1) != 1 {
        t.Fail()
    }
    
    t.Skip()
  
}

func TestGcdTest (t *testing.T){
    if GcdTest(1, 2, -2) {
        t.Fail()
    
    }
    
    if !GcdTest(5, 3, 2){
        t.Fail()
     
    }
    
    if GcdTest(5, 2, -2){
        t.Fail()
     
    }
    
    if !GcdTest(6, 2, 2, 2){
        t.Fail()
     
    }
  
    t.Skip()
  
}