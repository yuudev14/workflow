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
      <button className={cn('flex gap-4 p-2 bg-muted text-left hover:bg-accent', buttonClass)} onClick={onClick}>
        <div className={cn('aspect-square p-3 bg-primary text-primary-foreground', iconClass)} style={iconBgColor ? {
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