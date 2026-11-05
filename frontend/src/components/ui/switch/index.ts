import { cva, type VariantProps } from "class-variance-authority";

export { default as Switch } from "./Switch.vue";

export const switchVariants = cva(
  "peer inline-flex shrink-0 items-center rounded-full border-2 border-transparent transition-all outline-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] disabled:cursor-not-allowed disabled:opacity-50",
  {
    variants: {
      variant: {
        default: "data-[state=checked]:bg-primary data-[state=unchecked]:bg-accent",
        success: "data-[state=checked]:bg-success data-[state=unchecked]:bg-input dark:data-[state=unchecked]:bg-input/80",
        destructive: "data-[state=checked]:bg-destructive data-[state=unchecked]:bg-accent",
        outline: "border-border data-[state=checked]:bg-primary data-[state=unchecked]:bg-background",
      },
      size: {
        default: "h-6 w-11",
        sm: "h-4 w-7",
        lg: "h-6 w-10",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
);

export const switchThumbVariants = cva(
  "pointer-events-none block rounded-full ring-0 shadow-sm transition-transform duration-200 data-[state=checked]:translate-x-full data-[state=unchecked]:translate-x-0",
  {
    variants: {
      variant: {
        default: "data-[state=unchecked]:bg-muted-foreground data-[state=checked]:bg-primary-foreground",
        success: "bg-background dark:data-[state=unchecked]:bg-foreground dark:data-[state=checked]:bg-success-foreground",
        destructive: "data-[state=unchecked]:bg-muted-foreground data-[state=checked]:bg-destructive-foreground",
        outline: "bg-foreground dark:data-[state=checked]:bg-primary-foreground",
      },
      size: {
        default: "size-5",
        sm: "size-3",
        lg: "size-5",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
);

export type SwitchVariants = VariantProps<typeof switchVariants>;
export type SwitchThumbVariants = VariantProps<typeof switchThumbVariants>;
