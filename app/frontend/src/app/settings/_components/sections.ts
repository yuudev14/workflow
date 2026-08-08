import { KeyRound, ScrollText, ShieldCheck, Users, UsersRound, type LucideIcon } from "lucide-react";

/**
 * The settings area's own navigation. These used to sit in the main sidebar as
 * an "Admin" block; they live here so operational nav stays about running the
 * platform and administration is somewhere you go on purpose.
 *
 * Every section needs settings.read - the backend is the real boundary, this
 * only decides what is worth showing.
 */
export type SettingsSection = {
  title: string;
  url: string;
  icon: LucideIcon;
  description: string;
};

export const SETTINGS_SECTIONS: SettingsSection[] = [
  {
    title: "Users",
    url: "/settings/users",
    icon: Users,
    description: "Accounts, their roles, and password resets. Deactivate instead of deleting.",
  },
  {
    title: "Roles",
    url: "/settings/roles",
    icon: ShieldCheck,
    description: "The permission matrix each role grants. Builtin roles cannot be edited.",
  },
  {
    title: "Teams",
    url: "/settings/teams",
    icon: UsersRound,
    description: "Group people and grant roles to everyone in the group at once.",
  },
  {
    title: "Providers",
    url: "/settings/providers",
    icon: KeyRound,
    description: "Single sign-on sources. Users are provisioned and role-synced on first login.",
  },
  {
    title: "Audit",
    url: "/settings/audit",
    icon: ScrollText,
    description: "Every administrative change, who made it, and when. Read-only by design.",
  },
];
