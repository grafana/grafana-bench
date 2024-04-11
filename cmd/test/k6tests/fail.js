import { check } from 'k6'

export let options = {
   thresholds: {
        checks: ["rate >= 1"]
   }
}

export default function(){
        check({}, {
                "failed": () => false
        })
}