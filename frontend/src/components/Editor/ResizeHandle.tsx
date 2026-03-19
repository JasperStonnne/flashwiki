import { useCallback, useRef, useEffect } from "react";
import styles from "./ResizeHandle.module.css";

interface ResizeHandleProps {
  onResize: (widthPercent: number) => void;
  children: React.ReactNode;
  initialWidth?: number;
}

export function ResizeHandle({
  onResize,
  children,
  initialWidth = 100,
}: ResizeHandleProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const isDragging = useRef(false);
  const startX = useRef(0);
  const startWidth = useRef(0);
  const currentSide = useRef<"left" | "right">("right");

  const handleMouseDown = useCallback(
    (e: React.MouseEvent, side: "left" | "right") => {
      e.preventDefault();
      e.stopPropagation();
      isDragging.current = true;
      startX.current = e.clientX;
      currentSide.current = side;

      const container = containerRef.current;
      if (container) {
        startWidth.current = container.offsetWidth;
      }

      document.body.style.cursor = "ew-resize";
      document.body.style.userSelect = "none";
    },
    []
  );

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isDragging.current || !containerRef.current) return;

      const parent = containerRef.current.parentElement;
      if (!parent) return;

      const parentWidth = parent.offsetWidth;
      const deltaX = e.clientX - startX.current;
      const multiplier = currentSide.current === "right" ? 1 : -1;
      const newWidth = startWidth.current + deltaX * multiplier * 2;
      const newWidthPercent = Math.max(10, Math.min(100, (newWidth / parentWidth) * 100));

      onResize(newWidthPercent);
    };

    const handleMouseUp = () => {
      if (isDragging.current) {
        isDragging.current = false;
        document.body.style.cursor = "";
        document.body.style.userSelect = "";
      }
    };

    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);

    return () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
    };
  }, [onResize]);

  return (
    <div
      ref={containerRef}
      className={styles.container}
      style={{ width: `${initialWidth}%` }}
    >
      <div
        className={styles.handleLeft}
        onMouseDown={(e) => handleMouseDown(e, "left")}
      />
      {children}
      <div
        className={styles.handleRight}
        onMouseDown={(e) => handleMouseDown(e, "right")}
      />
    </div>
  );
}
