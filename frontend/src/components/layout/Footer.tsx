import React from 'react';
import { Building2, ShieldCheck, HeartHandshake } from 'lucide-react';

export const Footer: React.FC = () => {
  return (
    <footer className="bg-white brutal-border-t-4 mt-auto">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        <div className="flex flex-col md:flex-row items-center justify-between gap-4 text-xs font-bold text-black">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 bg-black text-white flex items-center justify-center font-black">
              <Building2 className="w-4 h-4" />
            </div>
            <div>
              <p className="font-black uppercase tracking-wider text-black">АО СЗ «ДСК»</p>
              <p className="text-[11px] text-neutral-500 font-semibold">Интеллектуальная платформа продаж недвижимости</p>
            </div>
          </div>

          <div className="flex items-center gap-6 text-black">
            <span className="flex items-center gap-1.5">
              <ShieldCheck className="w-4 h-4 text-black" />
              Официальные цены застройщика
            </span>
            <span className="flex items-center gap-1.5">
              <HeartHandshake className="w-4 h-4 text-black" />
              Прямой диалог с отделом продаж
            </span>
          </div>

          <p className="text-neutral-500 text-center md:text-right font-semibold">
            © {new Date().getFullYear()} АО СЗ «ДСК».
          </p>
        </div>
      </div>
    </footer>
  );
};
