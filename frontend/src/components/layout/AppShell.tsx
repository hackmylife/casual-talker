import { Outlet } from 'react-router'
import { Header } from './Header'
import { BottomNav } from './BottomNav'

export function AppShell() {
  return (
    <div className="flex flex-col min-h-svh">
      <Header />
      <main className="flex-1 pb-16">
        <Outlet />
      </main>
      <BottomNav />
    </div>
  )
}
