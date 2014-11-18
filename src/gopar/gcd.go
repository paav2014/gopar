package main


// simple binary gcd algorithm
func Gcd (a int, b int) int{
    if a< 0 {
        a = -a
    }
    if b< 0 {
        b = -b
    }
    if a == 0 { 
        return b
    }
    if b == 0 {
        return a
    }
    
    if a%2 == 0 && b%2==0 {
        return 2*Gcd(a/2, b/2)
    }
    if a%2 == 0 {
        return Gcd(a/2, b)
    }
    if b%2 == 0{
        return Gcd(a, b/2)
    }
    if a >= b {
        return Gcd((a-b)/2, b)
    }
    return Gcd((b-a)/2, a)
}

func GcdTest (c int, factors ...int)bool{
    if len(factors) < 1{
        return false    
    }
    
    l := factors[0]
    for i := 1; i < len(factors); i ++ {
        l := Gcd(l, factors[i])
        if i+1 == len(factors) {
            return c%l == 0
        }
    }
     
    return false  
}