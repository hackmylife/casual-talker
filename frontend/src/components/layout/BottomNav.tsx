import { NavLink } from 'react-router'
import { Home, BookOpen } from 'lucide-react'

const tabs = [
  { to: '/', label: 'ホーム', icon: Home },
  { to: '/history', label: '履歴', icon: BookOpen },
]

export function BottomNav() {
  return (
    <nav className="fixed bottom-0 left-0 right-0 flex border-t border-neutral-100 bg-white">
      {tabs.map(({ to, label, icon: Icon }) => (
        <NavLink
          key={to}
          to={to}
          end
          className={({ isActive }) =>
            [
              'flex flex-1 flex-col items-center gap-1 py-2 text-xs transition-colors',
              isActive
                ? 'text-primary-500'
                : 'text-neutral-600 hover:text-primary-500',
            ].join(' ')
          }
        >
          <Icon size={22} strokeWidth={1.8} />
          <span>{label}</span>
        </NavLink>
      ))}
    </nav>
  )
}
