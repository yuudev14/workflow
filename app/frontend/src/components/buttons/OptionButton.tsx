import { cn } from '@/lib/utils'
import { LucideProps } from 'lucide-react'
import React from 'react'

const OptionButton: React.FC<{
  Icon: React.ForwardRefExoticComponent<Omit<LucideProps, "ref"> & React.RefAttributes<SVGSVGElement>>,
  buttonClass?: string,
  iconClass?: string,
  iconBgColor?: string,
  iconTextColor?: string,
  onClick?: React.MouseEventHandler<HTMLButtonElement>
  children: React.ReactNode,
}> = ({
  Icon,
  buttonClass,
  iconClass,
  iconBgColor,
  iconTextColor,
  onClick,
  children
}) => {
    return (
      <button className={cn('flex items-center gap-3 rounded-md border border-line bg-card p-2.5 text-left transition-colors hover:border-line-strong hover:bg-paper-sunken', buttonClass)} onClick={onClick}>
        <div className={cn('flex aspect-square items-center justify-center rounded-lg bg-signal-soft p-2.5 text-signal-text [&_svg]:size-4', iconClass)} style={iconBgColor ? {
          backgroundColor: iconBgColor
        } : undefined}>
          <Icon style={iconTextColor ? {
            color: iconTextColor
          } : undefined} />
        </div>

        {children}


      </button>
    )
  }

export default OptionButton