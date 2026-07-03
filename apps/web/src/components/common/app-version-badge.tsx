import { ExternalLink, GitBranch } from 'lucide-react'

import { badgeVariants } from '@/components/ui/badge'
import { buttonVariants } from '@/components/ui/button'
import {
  Popover,
  PopoverContent,
  PopoverDescription,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  APP_UPDATE_COMMAND,
  appCommitSha,
  appCommitShortSha,
  appVersionLabel,
  formatAppVersion,
  getAppCommitUrl,
  getUpstreamDevelopCompareUrl,
} from '@/lib/app-version'
import { cn } from '@/lib/utils'

type AppVersionBadgeProps = {
  className?: string
  version?: string | null
}

export function AppVersionBadge({ className, version = appVersionLabel }: AppVersionBadgeProps) {
  const label = formatAppVersion(version)
  const appCommitUrl = getAppCommitUrl()
  const compareUrl = getUpstreamDevelopCompareUrl()

  return (
    <Popover>
      <PopoverTrigger
        aria-label={`前端版本 ${label}，查看本地构建信息`}
        className={cn(
          badgeVariants({ variant: 'outline' }),
          'cursor-pointer border-primary/25 bg-background/70 font-mono text-[0.68rem] text-muted-foreground hover:bg-muted hover:text-foreground',
          className,
        )}
        title={`前端版本 ${label}，查看本地构建信息`}
      >
        {label}
      </PopoverTrigger>
      <PopoverContent align="end" className="w-80 gap-3 text-xs">
        <PopoverHeader>
          <PopoverTitle className="flex items-center gap-2 text-sm">
            <GitBranch className="size-3.5 text-primary" />
            本地构建版本
          </PopoverTitle>
          <PopoverDescription>
            前端不直接请求 GitHub API；如需确认是否落后 develop，可打开 GitHub 对比页。
          </PopoverDescription>
        </PopoverHeader>

        <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-muted-foreground">
          <dt>当前版本</dt>
          <dd className="font-mono text-foreground">{label}</dd>
          <dt>当前提交</dt>
          <dd className="font-mono text-foreground">
            {appCommitUrl ? (
              <a
                className="underline-offset-4 hover:underline"
                href={appCommitUrl}
                rel="noreferrer"
                target="_blank"
              >
                {appCommitShortSha}
              </a>
            ) : (
              appCommitShortSha
            )}
          </dd>
          <dt>完整提交</dt>
          <dd className="break-all font-mono text-foreground">{appCommitSha || 'unknown'}</dd>
        </dl>

        <div className="rounded-md border border-border bg-muted/40 p-2">
          <div className="font-medium text-foreground">同步 develop</div>
          <code className="mt-1 block break-all font-mono text-[0.68rem] text-foreground">
            {APP_UPDATE_COMMAND}
          </code>
        </div>

        <a
          className={cn(buttonVariants({ variant: 'outline', size: 'sm' }), 'w-full')}
          href={compareUrl}
          rel="noreferrer"
          target="_blank"
        >
          <ExternalLink className="size-3.5" />
          打开 GitHub 对比
        </a>
      </PopoverContent>
    </Popover>
  )
}
