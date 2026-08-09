/** @type {import('tailwindcss').Config} */
export default {content:['./index.html','./src/**/*.{ts,tsx}'],theme:{extend:{colors:{brand:{primary:'var(--brand-primary)',secondary:'var(--brand-secondary)',accent:'var(--brand-accent)'}},boxShadow:{panel:'0 12px 35px rgba(15,23,42,.08)'}}},plugins:[]};
