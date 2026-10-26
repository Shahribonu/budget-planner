import { useRouter } from 'vue-router'

export function useBack() {
  const router = useRouter()
  return () => {
    if (window.history.state?.back) router.back()
    else void router.replace({ name: 'home' })
  }
}
